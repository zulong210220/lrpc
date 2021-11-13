package client

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	goproto "github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/proto"
	"github.com/zulong210220/lrpc/utils"

	jsoniter "github.com/json-iterator/go"

	"github.com/zulong210220/lrpc/consts"
	"github.com/zulong210220/lrpc/context"
	"github.com/zulong210220/lrpc/lcode"
	"github.com/zulong210220/lrpc/log"
	"github.com/zulong210220/lrpc/rpc"
)

type Call struct {
	Seq           uint64
	ServiceMethod string
	TraceId       string
	Args          lcode.IMessage
	Reply         lcode.IMessage
	Error         error
	Done          chan *Call
}

func (c *Call) done() {
	c.Done <- c
}

type Client struct {
	cc      net.Conn
	opt     *rpc.Option
	sending sync.Mutex
	header  lcode.Header
	mu      sync.Mutex
	seq     uint64
	pending map[uint64]*Call
	closing int32 // 关闭 就表示不可用
}

var (
	_ io.Closer = (*Client)(nil)

	ErrShutdown = errors.New("connection is shutdown")
)

const (
	StatusClosing = 1
)

var (
	limitedPool = utils.NewLimitedPool(consts.BufferPoolSizeMin, consts.BufferPoolSizeMax)

	bufferPool = sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}
)

func GetBuffer() *bytes.Buffer {
	buffer := bufferPool.Get().(*bytes.Buffer)
	return buffer
}

func PutBuffer(buffer *bytes.Buffer) {
	buffer.Reset()
	bufferPool.Put(buffer)
}

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
	return atomic.LoadInt32(&c.closing) != StatusClosing
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

	atomic.StoreInt32(&c.closing, StatusClosing)

	for _, ca := range c.pending {
		ca.Error = err
		ca.done()
	}
}

func (c *Client) Read(msg *lcode.Message) error {
	fun := "Client.Read"
	var data = make([]byte, 4)
	n, err := c.cc.Read(data)
	if err != nil {
		log.Errorf("CR", "%s connection total n:%d failed err:%v", fun, n, err)
		if err == io.EOF {
			// TODO
			return err
		}
		return err
	}

	total := binary.BigEndian.Uint32(data)

	if total > consts.BufferPoolSizeMax {
		data = make([]byte, total)
	} else {
		bb := limitedPool.Get(int(total))
		if len(*bb) > int(total) {
			data = (*bb)[:int(total)]
		} else {
			data = *bb
		}
		defer limitedPool.Put(bb)
	}

	n, err = c.cc.Read(data)
	if err != nil {
		log.Errorf("JCR", "%s connection data n:%d failed err:%v", fun, n, err)
		if err == io.EOF {
			// TODO
			return err
		}
	}

	err = msg.Unpack(data)
	return err
}

func (c *Client) Decode(b []byte, argvi interface{}) error {
	switch c.opt.CodecType {
	case lcode.GobType:
		return gob.NewDecoder(bytes.NewBuffer(b)).Decode(argvi)
	case lcode.JsonType:
		return jsoniter.Unmarshal(b, argvi)
	case lcode.ProtoType:
		proto.Unmarshal(b, argvi.(lcode.IMessage))
	case lcode.GoProtoType:
		return goproto.Unmarshal(b, argvi.(lcode.IMessage))
	}
	return nil
}

func (c *Client) receive() {
	var (
		err error
		fun = "Client.receive"
	)
	for err == nil {
		msg := &lcode.Message{H: &lcode.Header{}}
		err = c.Read(msg)
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
			err = c.Decode(msg.B, ca.Reply)
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

	if f == false {
		err := fmt.Errorf("%s invalid codec type %s", fun, opt.CodecType)
		log.Errorf("", "%s rpc client codec err:%v", fun, err)
		return nil, err
	}

	//err := json.NewEncoder(conn).Encode(opt)
	var err error
	dataBuf := bytes.NewBuffer([]byte{})
	err = jsoniter.NewEncoder(dataBuf).Encode(opt)
	if err != nil {
		log.Errorf("", "%s rpc client options failed err:%v", fun, err)
		return nil, err
	}
	pkg := dataBuf.Bytes()

	dataBuf = bytes.NewBuffer([]byte{})
	n := uint16(len(pkg))
	err = binary.Write(dataBuf, binary.BigEndian, n)
	if err != nil {
		log.Errorf("", " binary.Write len Option failed err:%v", err)
		return nil, err
	}

	_, err = conn.Write(dataBuf.Bytes())
	if err != nil {
		log.Errorf("", "%s rpc client options failed write n:%d err:%v", fun, n, err)
		_ = conn.Close()
		return nil, err
	}

	_, err = conn.Write(pkg)
	if err != nil {
		log.Errorf("", "%s rpc client options failed write n:%d err:%v", fun, n, err)
		_ = conn.Close()
		return nil, err
	}

	return newClientCodec(conn, opt), nil
}

func newClientCodec(cc net.Conn, opt *rpc.Option) *Client {
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

func (c *Client) Encode(body interface{}) []byte {
	fun := "Client.Encode"
	// TODO case type

	var (
		err    error
		bs     []byte
		buffer *bytes.Buffer
	)
	switch c.opt.CodecType {
	case lcode.GobType:
		buffer = GetBuffer()
		err = gob.NewEncoder(buffer).Encode(body)
		bs = buffer.Bytes()
	case lcode.JsonType:
		buffer = GetBuffer()
		err = jsoniter.NewEncoder(buffer).Encode(body)
		bs = buffer.Bytes()
	case lcode.ProtoType:
		bs, err = proto.Marshal(body.(lcode.IMessage))
	case lcode.GoProtoType:
		bs, err = goproto.Marshal(body.(lcode.IMessage))
	}
	if buffer != nil {
		PutBuffer(buffer)
	}

	if err != nil {
		log.Errorf("CE", "%s rpc codec: json Marshal failed error :%v", fun, err)
		return nil
	}
	return bs
}

func (c *Client) Write(h *lcode.Header, body interface{}) (err error) {
	fun := "Conn.Write"
	defer func() {
		if err != nil {
			_ = c.cc.Close()
		}
	}()

	bs := c.Encode(body)

	var n int
	msg := &lcode.Message{}
	msg.H = h
	msg.B = bs

	bs, err = msg.Pack()
	if err != nil {
		return
	}

	dataBuf := GetBuffer()
	err = binary.Write(dataBuf, binary.BigEndian, uint32(len(bs)))
	if err != nil {
		log.Errorf("CW", "%s binary Write len buffer:%v", fun, err)
		return
	}

	tbs := dataBuf.Bytes()
	PutBuffer(dataBuf)

	n, err = c.cc.Write(tbs)
	if err != nil {
		log.Errorf("CW", "%s rpc codec: json error write : %d total :%v", fun, n, err)
		return
	}

	n, err = c.cc.Write(bs)
	if err != nil {
		log.Errorf("CW", "%s rpc codec: json error write : %d buffer :%v", fun, n, err)
	}

	return
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
	c.header.TraceId = ca.TraceId

	err = c.Write(&c.header, ca.Args)
	//fmt.Println("aaa", c.header, ca.Args, err)
	if err != nil {
		ca := c.removeCall(seq)
		if ca != nil {
			ca.Error = err
			ca.done()
		}
	}
}

func (c *Client) Do(traceId, sm string, args, reply lcode.IMessage, done chan *Call) *Call {
	fun := "Client.Do"
	if done == nil {
		done = make(chan *Call, 16)
	} else if cap(done) == 0 {
		log.Errorf("", "%s rpc client done channel is unbuffered", fun)
	}
	ca := &Call{
		ServiceMethod: sm,
		TraceId:       traceId,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}

	c.send(ca)
	return ca
}

func (c *Client) Call(ctx *context.Context, sm string, args, reply lcode.IMessage) error {
	// send to server
	ca := c.Do(context.GetTraceId(ctx), sm, args, reply, make(chan *Call, 1))

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
