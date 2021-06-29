package rpc

import (
	"encoding/json"
	"fmt"
	"io"
	"lrpc/lcode"
	"net"
	"reflect"
	"sync"

	"github.com/sirupsen/logrus"
)

const (
	MagicNumber = 0x3bef5c
)

type Option struct {
	MagicNumber int
	CodecType   lcode.Type
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   lcode.GobType,
}

// | Option{MagicNumber: xxx, CodecType: xxx}  | Header{ServiceMethod ...} | Body interface{} |
// | <------      固定 JSON 编码      ------>  | <-------   编码方式由 CodeType 决定   ------->|
// | Option | Header1 | Body1 | Header2 | Body2 | ...

type Server struct{}

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
			logrus.Errorf("%s rpc server accept failed err:%v", fun, err)
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
	logrus.Infof("%s opt:%+v", fun, opt)
	if err != nil {
		logrus.Errorf("%s rpc server options error:%v", fun, err)
		return
	}

	if opt.MagicNumber != MagicNumber {
		logrus.Errorf("%s rpc server invalid magic number %x", fun, opt.MagicNumber)
		return
	}

	f := lcode.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		logrus.Errorf("%s rpc server invalid codec type %s", fun, opt.CodecType)
		return
	}

	s.serveCodec(f(conn))
}

var (
	invalidRequest = struct{}{}
)

func (s *Server) serveCodec(cc lcode.Codec) {
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
		go s.handleRequest(cc, req, sending, wg)
	}
	wg.Wait()
	_ = cc.Close()
}

type request struct {
	h            *lcode.Header
	argv, replyv reflect.Value
}

func (s *Server) readRequestHeader(cc lcode.Codec) (*lcode.Header, error) {
	fun := "Server.readRequestHeader"
	var h lcode.Header

	err := cc.ReadHeader(&h)
	if err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			logrus.Errorf("%s rpc server read header error:%v", fun, err)
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

	// TODO
	req.argv = reflect.New(reflect.TypeOf(""))
	err = cc.ReadBody(req.argv.Interface())
	if err != nil {
		logrus.Errorf("%s rpc server read argv failed err:%v", fun, err)
	}
	return req, err
}

func (s *Server) sendResponse(cc lcode.Codec, h *lcode.Header, body interface{}, sending *sync.Mutex) {
	fun := "Server.sendResponse"

	sending.Lock()
	defer sending.Unlock()
	err := cc.Write(h, body)
	if err != nil {
		logrus.Errorf("%s rpc server write response failed error:%v", fun, err)
	}

}

func (s *Server) handleRequest(cc lcode.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	fun := "Server.handleRequest"
	defer wg.Done()
	logrus.Info(fun, " : ", req.h, " : ", req.argv.Elem())

	req.replyv = reflect.ValueOf(fmt.Sprintf("lll rpc resp %d", req.h.Seq))
	s.sendResponse(cc, req.h, req.replyv.Interface(), sending)
}

/* vim: set tabstop=4 set shiftwidth=4 */
