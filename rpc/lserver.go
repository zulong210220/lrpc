package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zulong210220/lrpc/utils"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/zulong210220/lrpc/consts"
	"github.com/zulong210220/lrpc/lcode"
	"github.com/zulong210220/lrpc/log"
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

type Config struct {
	EtcdAddr    []string
	EtcdTimeout int
	ServerName  string
}

type Server struct {
	serviceMap    sync.Map
	client        *clientv3.Client
	leaseID       clientv3.LeaseID //租约ID
	keepAliveChan <-chan *clientv3.LeaseKeepAliveResponse
	ln            net.Listener
	name          string
	endpoint      string
	stop          chan error
	watchServers  []string
}

func NewServer() *Server {
	s := &Server{}
	return s
}

func (s *Server) Init(c *Config) {
	fun := "Server.Init"
	var err error
	config := clientv3.Config{
		Endpoints:   c.EtcdAddr,
		DialTimeout: time.Duration(c.EtcdTimeout) * time.Second,
	}
	s.name = c.ServerName

	s.client, err = clientv3.New(config)
	if err != nil {
		log.Errorf("%s err:%v", fun, err)
		return
	}
	s.ln, _ = net.Listen("tcp", ":0")

	lip, _ := utils.ExternalIP()
	s.endpoint = lip.String() + ":" + s.getListenPort()
	s.stop = make(chan error)
}

func (s *Server) getListenPort() string {
	la := s.ln.Addr().String()
	ss := strings.Split(la, ":")
	if len(ss) == 0 {
		return ""
	}
	return ss[len(ss)-1]
}

func (s *Server) getEtcdKey() string {
	return fmt.Sprintf("%s/%s/%s", consts.DefaultRegPath, s.name, s.endpoint)
}

func (s *Server) getEtcdValue() string {
	return strconv.Itoa(int(time.Now().Unix()))
}

func (s *Server) registryEtcd() error {
	fun := "Server.registryEtcd"
	ctx := context.Background()
	key := s.getEtcdKey()
	value := s.getEtcdValue()

	//创建一个新的租约，并设置ttl时间
	resp, err := s.client.Grant(context.Background(), consts.DefaultRegLease)
	if err != nil {
		log.Errorf("", "%s client.Grant failed err:%v", fun, err)
		return err
	}

	ps, err := s.client.Put(ctx, key, value, clientv3.WithLease(resp.ID))
	if err != nil {
		log.Errorf("", "%s client.Put ps:%+v failed err:%v", fun, ps, err)
		return err
	}

	//设置续租 定期发送需求请求
	//KeepAlive使给定的租约永远有效。 如果发布到channel的keepalive响应没有立即被使用，
	// 则租约客户端将至少每秒钟继续向etcd服务器发送保持活动请求，直到获取最新的响应为止。
	//etcd client会自动发送ttl到etcd server，从而保证该租约一直有效
	leaseRespChan, err := s.client.KeepAlive(context.Background(), resp.ID)
	if err != nil {
		log.Errorf("", "%s client.KeepAlive failed err:%v", fun, err)
		return err
	}

	s.leaseID = resp.ID
	s.keepAliveChan = leaseRespChan

	s.selectLoop()
	return err
}

// revoke 注销服务
func (s *Server) revoke() error {
	fun := "Server.revoke"
	//撤销租约
	if _, err := s.client.Revoke(context.Background(), s.leaseID); err != nil {
		log.Errorf("", "%s client.Revoke failed err:%v", fun, err)
		return err
	}
	return s.client.Close()
}

func (s *Server) Stop() {
	s.stop <- nil
}

// https://github.com/golang/go/issues/27707

var DefaultServer = NewServer()

func (s *Server) Accept(ln net.Listener) {
	fun := "Server.Accept"
	for {
		conn, err := ln.Accept()
		fmt.Println("Accept", conn.LocalAddr(), conn.LocalAddr())
		if err != nil {
			log.Errorf("", "%s rpc server accept failed err:%v", fun, err)
			return
		}

		go s.ServeConn(conn)
	}
}

func (s *Server) selectLoop() {
	for {
		select {
		case err := <-s.stop:
			log.Error("", "Server.selectLoop stop failed err:", err)
			s.revoke()
			return
		case <-s.client.Ctx().Done():
			log.Error("", "server closed")
			// select keep alive chan 要在etcd初始化之后才有效
		case _, ok := <-s.keepAliveChan:
			if !ok {
				s.revoke()
				return
			} else {
				//log.Infof("", "Recv reply from service: %s, ttl:%d", s.name, ka.TTL)
			}
		}
	}
}

func (s *Server) Run() {
	go s.registryEtcd()
	s.Accept(s.ln)
}

func Accept(ln net.Listener) {
	DefaultServer.Accept(ln)
}

func (s *Server) ServeConn(conn io.ReadWriteCloser) {
	fun := "Server.ServeConn"
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("Close", "Server.ServeConn Close failed err:%v", err)
		}
	}()
	var opt Option
	//data, err := ioutil.ReadAll(conn)

	var data = make([]byte, consts.HandleshakeBufLen)
	n, err := conn.Read(data)

	// also block
	// var buf bytes.Buffer
	// nn, err := io.Copy(&buf, conn)
	if err != nil {
		log.Errorf("", "%s ioutil.ReadAll options error:%v", fun, err)
		return
	}
	//err := json.NewDecoder(conn).Decode(&opt)
	n = 0
	for n < len(data) {
		if data[n] == '}' {
			break
		}
		n++
	}
	data = data[:n+1]

	err = json.Unmarshal(data, &opt)
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
	//fun := "Server.serveCodec"
	sending := &sync.Mutex{}
	wg := &sync.WaitGroup{}

	for {
		// wait 偶尔阻塞在此
		req, err := s.readRequest(cc)
		if err != nil {
			if req == nil {
				wg = nil
				break
			}

			req.h.Error = err.Error()
			s.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}

		wg.Add(1)
		go s.handleRequest(cc, req, sending, wg, opt.HandleTimeout)
	}
	if wg != nil {
		wg.Wait()
	}
}

type request struct {
	h            *lcode.Header
	argv, replyv reflect.Value
	mType        *methodType
	svc          *service
}

func (r *request) Header() *lcode.Header {
	h := &lcode.Header{
		ServiceMethod: r.h.ServiceMethod,
		Seq:           r.h.Seq,
		Error:         r.h.Error,
	}
	return h
}

func (s *Server) readRequest(cc lcode.Codec) (*request, error) {
	fun := "Server.readRequest"

	msg := &lcode.Message{H: &lcode.Header{}}
	err := cc.Read(msg)
	if err != nil {
		return nil, err
	}

	req := &request{h: msg.H}
	req.svc, req.mType, err = s.findService(msg.H.ServiceMethod)
	if err != nil {
		log.Errorf("", "%s findService failed serviceMethod:%s err:%v", fun, msg.H.ServiceMethod, err)
		return req, err
	}

	// TODO
	req.argv = req.mType.newArgv()
	req.replyv = req.mType.newReplyv()
	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}

	//err = cc.ReadBody(argvi)
	err = cc.Decode(msg.B, argvi)
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
		return
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

	// serviceName.methodName
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
