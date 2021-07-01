package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"lrpc/lcode"
	"lrpc/log"
	"lrpc/rpc"
	"net"
	"sync"
	"sync/atomic"
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
		var h lcode.Header
		err = c.cc.ReadHeader(&h)
		// TODO conn关闭会报错
		if err != nil {
			log.Errorf("%s ReadHeader failed err:%v", fun, err)
			break
		}

		ca := c.removeCall(h.Seq)
		switch {
		case ca == nil:
			err = c.cc.ReadBody(nil)
		case h.Error != "":
			ca.Error = errors.New(h.Error)
			err = c.cc.ReadBody(nil)
			ca.done()
		default:
			err = c.cc.ReadBody(ca.Reply)
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
		log.Errorf("%s rpc client codec err:%v", fun, err)
		return nil, err
	}

	err := json.NewEncoder(conn).Encode(opt)
	if err != nil {
		log.Errorf("%s rpc client options failed err:%v", fun, err)
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
	fun := "Dial"
	opt, err := parseOptions(opts...)
	if err != nil {
		log.Errorf("%s parseOptions failed err:%v", fun, err)
		return nil, err
	}

	conn, err := net.Dial(network, addr)
	if err != nil {
		log.Errorf("%s net.Dial network:%s addr:%s failed err:%v", fun, network, addr, err)
		return nil, err
	}

	defer func() {
		if c == nil {
			_ = conn.Close()
		}
	}()

	c, err = NewClient(conn, opt)
	return
}

func (c *Client) send(ca *Call) {
	fun := "Client.send"
	// 并发发送需要加锁
	c.sending.Lock()
	defer c.sending.Unlock()

	seq, err := c.registerCall(ca)
	if err != nil {
		log.Errorf("%s client registerCall failed err:%v", fun, err)
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
	log.Infof("%s Do enter done:%d", fun, cap(done))
	if done == nil {
		done = make(chan *Call, 16)
	} else if cap(done) == 0 {
		log.Errorf("%s rpc client done channel is unbuffered", fun)
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

func (c *Client) Call(sm string, args, reply interface{}) error {
	ca := <-c.Do(sm, args, reply, make(chan *Call, 1)).Done
	return ca.Error
}
