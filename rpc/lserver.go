package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"github.com/zulong210220/lrpc/consts"
	"github.com/zulong210220/lrpc/lcode"
	"github.com/zulong210220/lrpc/log"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"
)

const (
	MagicNumber = 0x3bef5c
)

type Option struct {
	MagicNumber    int
	CodecType      lcode.Type
	ConnectTimeout time.Duration // 客户端连接超时时间
	HandleTimeout  time.Duration // 服务端处理超时时间
}

var DefaultOption = &Option{
	MagicNumber:    MagicNumber,
	CodecType:      lcode.GobType,
	ConnectTimeout: 3 * time.Second,
}

// | Option{MagicNumber: xxx, CodecType: xxx}  | Header{ServiceMethod ...} | Body interface{} |
// | <------      固定 JSON 编码      ------>  | <-------   编码方式由 CodeType 决定   ------->|
// | Option | Header1 | Body1 | Header2 | Body2 | ...

type Server struct {
	serviceMap sync.Map
}

func NewServer() *Server {
	s := &Server{}
	return s
}

var DefaultServer = NewServer()

func (s *Server) Accept(ln net.Listener) {
	fun := "Server.Accept"
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Errorf("", "%s rpc server accept failed err:%v", fun, err)
			return
		}

		go s.ServeConn(conn)
	}
}

func Accept(ln net.Listener) {
	DefaultServer.Accept(ln)
}

func (s *Server) ServeConn(conn io.ReadWriteCloser) {
	fun := "Server.ServeConn"
	defer func() {
		_ = conn.Close()
	}()
	var opt Option
	err := json.NewDecoder(conn).Decode(&opt)
	log.Infof("", "%s opt:%+v", fun, opt)
	if err != nil {
		log.Errorf("", "%s rpc server options error:%v", fun, err)
		return
	}

	if opt.MagicNumber != MagicNumber {
		log.Errorf("", "%s rpc server invalid magic number %x", fun, opt.MagicNumber)
		return
	}

	f := lcode.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Errorf("", "%s rpc server invalid codec type %s", fun, opt.CodecType)
		return
	}

	s.serveCodec(f(conn), &opt)
}

var (
	invalidRequest = struct{}{}
)

func (s *Server) serveCodec(cc lcode.Codec, opt *Option) {
	sending := &sync.Mutex{}
	wg := &sync.WaitGroup{}

	for {
		req, err := s.readRequest(cc)
		if err != nil {
			if req == nil {
				break
			}

			req.h.Error = err.Error()
			s.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}

		wg.Add(1)
		go s.handleRequest(cc, req, sending, wg, opt.HandleTimeout)
	}
	wg.Wait()
	_ = cc.Close()
}

type request struct {
	h            *lcode.Header
	argv, replyv reflect.Value
	mType        *methodType
	svc          *service
}

func (s *Server) readRequestHeader(cc lcode.Codec) (*lcode.Header, error) {
	fun := "Server.readRequestHeader"
	var h lcode.Header

	err := cc.ReadHeader(&h)
	if err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Errorf("", "%s rpc server read header error:%v", fun, err)
		}
		return nil, err
	}
	return &h, nil
}

func (s *Server) readRequest(cc lcode.Codec) (*request, error) {
	fun := "Server.readRequest"
	h, err := s.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}

	req := &request{h: h}
	req.svc, req.mType, err = s.findService(h.ServiceMethod)
	if err != nil {
		log.Errorf("", "%s findService failed serviceMethod:%s err:%v", fun, h.ServiceMethod, err)
		return req, err
	}

	// TODO
	req.argv = req.mType.newArgv()
	req.replyv = req.mType.newReplyv()
	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}

	err = cc.ReadBody(argvi)
	if err != nil {
		log.Errorf("", "%s rpc server read argv failed err:%v", fun, err)
	}
	return req, err
}

func (s *Server) sendResponse(cc lcode.Codec, h *lcode.Header, body interface{}, sending *sync.Mutex) {
	fun := "Server.sendResponse"

	sending.Lock()
	defer sending.Unlock()
	err := cc.Write(h, body)
	if err != nil {
		log.Errorf("", "%s rpc server write response failed error:%v", fun, err)
	}

}

func (s *Server) handleRequest(cc lcode.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup, timeout time.Duration) {
	fun := "Server.handleRequest"
	defer wg.Done()
	log.Info("", fun, " : ", req.h, " : ", req.argv)

	called := make(chan struct{})
	send := make(chan struct{})

	go func() {
		// 此处真正执行代码逻辑
		err := req.svc.call(req.mType, req.argv, req.replyv)
		called <- struct{}{}
		if err != nil {
			req.h.Error = err.Error()
			s.sendResponse(cc, req.h, invalidRequest, sending)
			send <- struct{}{}
			return
		}

		s.sendResponse(cc, req.h, req.replyv.Interface(), sending)
		send <- struct{}{}
	}()

	if timeout == 0 {
		<-called
		<-send
	}

	select {
	case <-time.After(timeout):
		req.h.Error = fmt.Sprintf("rpc server: request handle timeout %s", timeout)
		s.sendResponse(cc, req.h, req.replyv.Interface(), sending)
	case <-called:
		<-send
	}
}

func (s *Server) Register(rcvr interface{}) error {
	sv := newService(rcvr)

	_, dup := s.serviceMap.LoadOrStore(sv.name, sv)
	if dup {
		log.Error("", "rpc service already registered: ", sv.name)
		return consts.ErrRegDup
	}
	return nil
}

func Register(rcvr interface{}) error {
	return DefaultServer.Register(rcvr)
}

func (s *Server) findService(sm string) (svc *service, mType *methodType, err error) {
	dot := strings.Index(sm, ".")
	if dot < 0 {
		err = errors.New("rpc server: service/method request ill-formed: " + sm)
		return
	}

	sn, mn := sm[:dot], sm[dot+1:]

	sv, ok := s.serviceMap.Load(sn)
	if !ok {
		err = errors.New("rpc server: can't find service: " + sn)
		return
	}

	svc = sv.(*service)
	mType = svc.method[mn]

	if mType == nil {
		err = errors.New("rpc server: can't find method: " + mn)
	}

	return
}

/* vim: set tabstop=4 set shiftwidth=4 */
