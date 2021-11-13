package rpc

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	goproto "github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/proto"
	jsoniter "github.com/json-iterator/go"
	"github.com/zulong210220/lrpc/consts"
	"github.com/zulong210220/lrpc/lcode"
	"github.com/zulong210220/lrpc/log"
	"github.com/zulong210220/lrpc/utils"
)

/*
	Conn {
	conn
	chan
	}
*/

// chan conn req->handle->resp

type Conn struct {
	state     int32
	fd        int
	workerNum int
	s         *Server
	conn      net.Conn
	opt       *Option
	reqChan   chan *request
	respChan  chan *response
	closeChan chan bool
	die       chan struct{}
}

const (
	DefaultHandlerNumber = 4

	StateRunninng = 1
	StateClosed   = 2
)

func NewConn(s *Server, conn net.Conn) *Conn {
	return &Conn{
		state:     StateRunninng,
		fd:        socketFD(conn),
		workerNum: DefaultHandlerNumber,
		s:         s,
		conn:      conn,
		reqChan:   make(chan *request, 64),
		respChan:  make(chan *response, 64),
		closeChan: make(chan bool, 64),
		die:       make(chan struct{}),
	}
}

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

func (c *Conn) Serve() {
	fun := "Conn.Serve"
	//defer func() {
	//	err := c.conn.Close()
	//	if err != nil {
	//		log.Errorf("Close", "Server.ServeConn Close failed err:%v", err)
	//	}
	//}()
	//data, err := ioutil.ReadAll(conn)

	err := c.preHandle()
	if err != nil {
		return
	}

	f := lcode.NewCodecFuncMap[c.opt.CodecType]
	if f == false {
		log.Errorf("", "%s rpc server invalid codec type %s", fun, c.opt.CodecType)
		return
	}
	c.startWorkers()
	go c.close()
	c.serveCodec()

}

func (c *Conn) preHandle() error {
	fun := "Server.preHandle"
	var data = make([]byte, 2)
	n, err := c.conn.Read(data)
	if err != nil {
		log.Errorf("", "%s rpc server read failed %v", fun, err)
		return err
	}

	total := binary.BigEndian.Uint16(data)
	data = make([]byte, total)
	n, err = c.conn.Read(data)
	if err != nil {
		log.Errorf("", "%s rpc server read failed %v", fun, err)
		return err
	}

	if uint16(n) != total {
		log.Warningf("ne", "%s rpc server read n:%d total:%d", fun, n, total)
	}

	opt := &Option{}
	err = jsoniter.Unmarshal(data, opt)
	if err != nil {
		log.Errorf("UnpackHeader", " Unmarshal Option failed err:%v", err)
		return err
	}

	// also block

	if opt.MagicNumber != MagicNumber {
		log.Errorf("", "%s rpc server invalid magic number %x", fun, opt.MagicNumber)
		return errors.New("Invalid MagicNumber")
	}

	if opt.HandleTimeout == 0 {
		opt.HandleTimeout = 3 * time.Second
	}

	c.opt = opt

	return err
}

// producer
func (c *Conn) serveCodec() {
	//fun := "Server.serveCodec"
	for {
		select {
		case <-c.closeChan:
			return
		default:
			// wait 偶尔阻塞在此
			req, err := c.readRequest()
			if err != nil {
				// close conn
				c.Close()
				return
				//req.h.Error = err.Error()
				////c.sendResponse(req.h, invalidRequest, sending)
				//resp := &response{
				//	h:    req.h,
				//	body: invalidRequest,
				//}
				//c.respChan <- resp
				//continue
			}
			c.reqChan <- req
			//go c.handleRequest(req, sending, wg, c.opt.HandleTimeout)
		}
	}
}

func (c *Conn) startWorkers() {
	for i := 0; i < c.workerNum; i++ {
		go c.loopHandleRequest(i + 1)
	}
	go c.handleResponse()
}

// comsumer
func (c *Conn) loopHandleRequest(i int) {
	for {
		select {
		case <-c.closeChan:
			return
		case req := <-c.reqChan:
			if req == nil {
				return
			}
			c.handleSingleRequest(req)
		}
	}
}

func (c *Conn) handleSingleRequest(req *request) {
	called := make(chan struct{})
	send := make(chan struct{})

	resp := &response{}
	timeout := c.opt.HandleTimeout

	go func() {
		// 此处真正执行代码逻辑
		err := req.svc.call(req.mType, req.argv, req.replyv)
		called <- struct{}{}
		if err != nil {
			req.h.Error = err.Error()

			resp.h = req.h
			resp.body = invalidRequest

			c.respChan <- resp
			send <- struct{}{}
			return
		}
		resp.h = req.h
		resp.body = req.replyv.Interface()

		c.respChan <- resp
		send <- struct{}{}
	}()

	/*
		if timeout == 0 {
			<-called
			<-send
			return
		}
	*/

	select {
	case <-time.After(timeout):
		req.h.Error = fmt.Sprintf("rpc server: request handle timeout %s", timeout)
		resp.h = req.h
		resp.body = req.replyv.Interface()
		c.respChan <- resp
	case <-called:
		<-send
	}
}

func (c *Conn) handleResponse() {
	fun := "Server.sendResponse"
	for {
		select {
		case <-c.closeChan:
			goto clear
		case resp := <-c.respChan:
			if resp == nil {
				break
			}

			traceId := resp.h.TraceId
			err := c.Write(resp.h, resp.body)
			if err != nil {
				log.Errorf(traceId, "%s rpc server write response failed error:%v", fun, err)
			}
		}
	}

clear:
	close(c.respChan)
	for resp := range c.respChan {
		if resp == nil {
			break
		}

		traceId := resp.h.TraceId
		err := c.Write(resp.h, resp.body)
		if err != nil {
			log.Errorf(traceId, "%s rpc server write response failed error:%v", fun, err)
		}
	}
	err := c.conn.Close()
	if err != nil {
		log.Errorf("Close", "Conn:%d close failed err:%v", c.fd, err)
	}
}

func (c *Conn) serveCodec0() {
	//fun := "Server.serveCodec"
	sending := &sync.Mutex{}
	wg := &sync.WaitGroup{}

	for {
		// wait 偶尔阻塞在此
		req, err := c.readRequest()
		if err != nil {
			if req == nil {
				wg = nil
				break
			}

			req.h.Error = err.Error()
			c.sendResponse(req.h, invalidRequest, sending)
			continue
		}

		wg.Add(1)
		go c.handleRequest(req, sending, wg, c.opt.HandleTimeout)
	}
	if wg != nil {
		wg.Wait()
	}
}

func (c *Conn) Read(msg *lcode.Message) error {
	fun := "Conn.Read"
	var data = make([]byte, 4)
	n, err := c.conn.Read(data)
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

	n, err = c.conn.Read(data)
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

func (c *Conn) readRequest() (*request, error) {
	fun := "Server.readRequest"

	req := &request{}
	msg := &lcode.Message{H: &lcode.Header{}}
	err := c.Read(msg)
	if err != nil {
		return req, err
	}
	traceId := msg.H.TraceId

	req.h = msg.H
	req.svc, req.mType, err = c.s.findService(msg.H.ServiceMethod)
	if err != nil {
		log.Errorf(traceId, "%s findService failed serviceMethod:%s err:%v", fun, msg.H.ServiceMethod, err)
		return req, err
	}

	// TODO
	req.argv = req.mType.newArgv()
	req.replyv = req.mType.newReplyv()
	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}

	err = c.Decode(msg.B, argvi)
	if err != nil {
		log.Errorf(traceId, "%s rpc server read argv failed err:%v", fun, err)
	}
	return req, err
}

func (c *Conn) Decode(b []byte, argvi interface{}) error {
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

func (c *Conn) handleRequest(req *request, sending *sync.Mutex, wg *sync.WaitGroup, timeout time.Duration) {
	//fun := "Server.handleRequest"
	defer wg.Done()

	called := make(chan struct{})
	send := make(chan struct{})

	go func() {
		// 此处真正执行代码逻辑
		err := req.svc.call(req.mType, req.argv, req.replyv)
		called <- struct{}{}
		if err != nil {
			req.h.Error = err.Error()
			c.sendResponse(req.h, invalidRequest, sending)
			send <- struct{}{}
			return
		}

		c.sendResponse(req.h, req.replyv.Interface(), sending)
		send <- struct{}{}
	}()

	if timeout == 0 {
		<-called
		<-send
		return
	}

	select {
	case <-time.After(timeout):
		req.h.Error = fmt.Sprintf("rpc server: request handle timeout %s", timeout)
		c.sendResponse(req.h, req.replyv.Interface(), sending)
	case <-called:
		<-send
	}
}

func (c *Conn) sendResponse(h *lcode.Header, body interface{}, sending *sync.Mutex) {
	fun := "Server.sendResponse"
	traceId := h.TraceId

	sending.Lock()
	defer sending.Unlock()
	err := c.Write(h, body)
	if err != nil {
		log.Errorf(traceId, "%s rpc server write response failed error:%v", fun, err)
	}
}

func (c *Conn) Encode(body interface{}) []byte {
	fun := "Conn.Encode"
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

func (c *Conn) Write(h *lcode.Header, body interface{}) (err error) {
	fun := "Conn.Write"
	defer func() {
		if err != nil {
			c.Close()
		}
	}()

	bs := c.Encode(body)

	var n int
	msg := &lcode.Message{}
	msg.H = h
	msg.B = bs
	traceId := h.TraceId

	bs, err = msg.Pack()
	if err != nil {
		return
	}

	dataBuf := GetBuffer()
	err = binary.Write(dataBuf, binary.BigEndian, uint32(len(bs)))
	if err != nil {
		log.Errorf(traceId, "%s binary Write len buffer:%v", fun, err)
		return
	}

	tbs := dataBuf.Bytes()
	PutBuffer(dataBuf)

	n, err = c.conn.Write(tbs)
	if err != nil {
		log.Errorf(traceId, "%s rpc codec: json error write : %d total :%v", fun, n, err)
		return
	}

	n, err = c.conn.Write(bs)
	if err != nil {
		log.Errorf(traceId, "%s rpc codec: json error write : %d buffer :%v", fun, n, err)
	}

	return
}

func (c *Conn) Close() {
	if atomic.LoadInt32(&c.state) == StateClosed {
		return
	}

	atomic.StoreInt32(&c.state, StateClosed)
	c.die <- struct{}{}
}

func (c *Conn) close() {
	<-c.die
	// 假定开启的goroutine不会超过65535
	for i := 0; i < 64; i++ {
		c.closeChan <- true
	}

}

func socketFD(conn net.Conn) int {
	//tls := reflect.TypeOf(conn.UnderlyingConn()) == reflect.TypeOf(&tls.Conn{})
	// Extract the file descriptor associated with the connection
	//connVal := reflect.Indirect(reflect.ValueOf(conn)).FieldByName("conn").Elem()
	tcpConn := reflect.Indirect(reflect.ValueOf(conn)).FieldByName("conn")
	//if tls {
	//	tcpConn = reflect.Indirect(tcpConn.Elem())
	//}
	fdVal := tcpConn.FieldByName("fd")
	pfdVal := reflect.Indirect(fdVal).FieldByName("pfd")
	return int(pfdVal.FieldByName("Sysfd").Int())
}
