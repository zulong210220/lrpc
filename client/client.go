package client

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zulong210220/lrpc/consts"
	"github.com/zulong210220/lrpc/lcode"
	"github.com/zulong210220/lrpc/log"
	"github.com/zulong210220/lrpc/rpc"
)

type Call struct {
	Seq           uint64
	ServiceMethod string
	Args          interface{}
	Reply         interface{}
	Error         error
	Done          chan *Call
}

func (c *Call) done() {
	c.Done <- c
}

type Client struct {
	cc       lcode.Codec
	opt      *rpc.Option
	sending  sync.Mutex
	header   lcode.Header
	mu       sync.Mutex
	seq      uint64
	pending  map[uint64]*Call
	closing  int32
	shutdown int32
}

var (
	_ io.Closer = (*Client)(nil)

	ErrShutdown = errors.New("connection is shutdown")
)

const (
	StatusClosing  = 1
	StatusShutdown = 1
)

func (c *Client) Close() error {
	//c.mu.Lock()
	//defer c.mu.Unlock()
	if c == nil {
		return nil
	}

	if atomic.LoadInt32(&c.closing) == StatusClosing {
		return ErrShutdown
	}

	atomic.StoreInt32(&c.closing, StatusClosing)
	return c.cc.Close()
}

func (c *Client) IsAvailable() bool {
	// TODO lock
	return atomic.LoadInt32(&c.shutdown) != StatusShutdown && atomic.LoadInt32(&c.closing) != StatusClosing
}

func (c *Client) registerCall(ca *Call) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.IsAvailable() {
		return 0, ErrShutdown
	}

	ca.Seq = c.seq
	c.pending[ca.Seq] = ca
	c.seq++
	return ca.Seq, nil
}

func (c *Client) removeCall(seq uint64) *Call {
	c.mu.Lock()
	defer c.mu.Unlock()

	ca := c.pending[seq]
	delete(c.pending, seq)

	return ca
}

func (c *Client) terminateCalls(err error) {
	c.sending.Lock()
	defer c.sending.Unlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	atomic.StoreInt32(&c.shutdown, StatusShutdown)

	for _, ca := range c.pending {
		ca.Error = err
		ca.done()
	}
}

func (c *Client) receive() {
	var (
		err error
		fun = "Client.receive"
	)
	for err == nil {
		msg := &lcode.Message{H: &lcode.Header{}}
		err = c.cc.Read(msg)
		// TODO conn关闭会报错
		if err != nil {
			log.Errorf("", "%s ReadHeader failed err:%v", fun, err)
			break
		}

		h := msg.H
		ca := c.removeCall(h.Seq)

		switch {
		case ca == nil:
		case h.Error != "":
			ca.Error = errors.New(h.Error)
			ca.done()
		default:
			//err = c.cc.ReadBody(ca.Reply)
			err = c.cc.Decode(msg.B, ca.Reply)
			if err != nil {
				ca.Error = fmt.Errorf("%s reading body err:%v", fun, err)
			}
			ca.done()
		}
	}

	c.terminateCalls(err)
}

func NewClient(conn net.Conn, opt *rpc.Option) (*Client, error) {
	fun := "NewClient"
	f := lcode.NewCodecFuncMap[opt.CodecType]

	if f == nil {
		err := fmt.Errorf("%s invalid codec type %s", fun, opt.CodecType)
		log.Errorf("", "%s rpc client codec err:%v", fun, err)
		return nil, err
	}

	//err := json.NewEncoder(conn).Encode(opt)
	data, err := json.Marshal(opt)
	if err != nil {
		log.Errorf("", "%s rpc client options failed err:%v", fun, err)
		_ = conn.Close()
		return nil, err
	}

	buf := make([]byte, consts.HandleshakeBufLen)
	var n int
	copy(buf, data)
	n, err = conn.Write(buf)
	if err != nil {
		log.Errorf("", "%s rpc client options failed write n:%d err:%v", fun, n, err)
		_ = conn.Close()
		return nil, err
	}

	return newClientCodec(f(conn), opt), nil
}

func newClientCodec(cc lcode.Codec, opt *rpc.Option) *Client {
	c := &Client{
		seq:     1,
		cc:      cc,
		opt:     opt,
		pending: make(map[uint64]*Call),
	}

	go c.receive()
	return c
}

func parseOptions(opts ...*rpc.Option) (*rpc.Option, error) {
	if len(opts) == 0 || opts[0] == nil {
		return rpc.DefaultOption, nil
	}

	// TODO
	if len(opts) != 1 {
		return nil, errors.New("number of option is more than 1")
	}

	opt := opts[0]
	opt.MagicNumber = rpc.DefaultOption.MagicNumber

	if opt.CodecType == "" {
		opt.CodecType = rpc.DefaultOption.CodecType
	}

	return opt, nil
}

func Dial(network, addr string, opts ...*rpc.Option) (c *Client, err error) {
	return dialTimeout(NewClient, network, addr, opts...)
}

func (c *Client) send(ca *Call) {
	if c == nil {
		if ca != nil {
			ca.done()
		}
		return
	}
	fun := "Client.send"
	// 并发发送需要加锁
	c.sending.Lock()
	defer c.sending.Unlock()

	seq, err := c.registerCall(ca)
	if err != nil {
		log.Errorf("", "%s client registerCall failed err:%v", fun, err)
		ca.Error = err
		ca.done()
		return
	}

	c.header.ServiceMethod = ca.ServiceMethod
	c.header.Seq = seq
	c.header.Error = ""

	err = c.cc.Write(&c.header, ca.Args)
	if err != nil {
		ca := c.removeCall(seq)
		if ca != nil {
			ca.Error = err
			ca.done()
		}
	}
}

func (c *Client) Do(sm string, args, reply interface{}, done chan *Call) *Call {
	fun := "Client.Do"
	if done == nil {
		done = make(chan *Call, 16)
	} else if cap(done) == 0 {
		log.Errorf("", "%s rpc client done channel is unbuffered", fun)
	}
	ca := &Call{
		ServiceMethod: sm,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}

	c.send(ca)
	return ca
}

func (c *Client) Call(ctx context.Context, sm string, args, reply interface{}) error {
	// send to server
	ca := c.Do(sm, args, reply, make(chan *Call, 1))

	// wait receive done
	// 可能存在server不响应的情况
	select {
	case <-ctx.Done():
		c.removeCall(ca.Seq)
		return fmt.Errorf("rpc client : call failed err:%s", ctx.Err().Error())
	case cd := <-ca.Done:
		return cd.Error
	}
}

type clientResult struct {
	client *Client
	err    error
}

type newClientFunc func(conn net.Conn, opt *rpc.Option) (c *Client, err error)

func dialTimeout(f newClientFunc, network, addr string, opts ...*rpc.Option) (c *Client, err error) {
	fun := "dialTimeout"
	opt, err := parseOptions(opts...)
	if err != nil {
		log.Errorf("%s parseOptions failed err:%v", fun, err)
		return nil, err
	}

	conn, err := net.DialTimeout(network, addr, opt.ConnectTimeout)
	if err != nil {
		log.Errorf("", "%s DialTimeout failed network:%s addr:%s err:%v", fun, network, addr, err)
		return nil, err
	}

	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()

	ch := make(chan clientResult)
	go func() {
		cli, err := f(conn, opt)
		ch <- clientResult{client: cli, err: err}
	}()

	// 阻塞式
	if opt.ConnectTimeout == 0 {
		result := <-ch
		return result.client, result.err
	}

	select {
	case <-time.After(opt.ConnectTimeout):
		return nil, fmt.Errorf("rpc client: connect timeout %s ", opt.ConnectTimeout)
	case result := <-ch:
		return result.client, result.err
	}

	return
}

// ----
func NewHTTPClient(conn net.Conn, opt *rpc.Option) (*Client, error) {
	_, _ = io.WriteString(conn, fmt.Sprintf("%s %s HTTP/1.0\n\n", consts.MethodConnect, consts.DefaultRpcPath))
	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{
		Method: "CONNECT",
	})

	if err == nil && resp.Status == consts.Connected {
		return NewClient(conn, opt)
	}

	if err == nil {
		err = errors.New("unexpected HTTP Response: " + resp.Status)
	}
	return nil, err
}

func DialHTTP(network, addr string, opts ...*rpc.Option) (*Client, error) {
	return dialTimeout(NewHTTPClient, network, addr, opts...)
}

func XDial(rpcAddr string, opts ...*rpc.Option) (*Client, error) {
	parts := strings.Split(rpcAddr, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("rpc client error: invalid format '%s', expect protocol@addr", rpcAddr)
	}

	protocol, addr := parts[0], parts[1]

	switch protocol {
	case consts.ProtocolHTTP:
		return DialHTTP("tcp", addr, opts...)
	default:
		return Dial(protocol, addr, opts...)
	}

	return nil, errors.New("invalid protocol")
}
