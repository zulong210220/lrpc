package rpc

import (
	"io"
	"github.com/zulong210220/lrpc/consts"
	"github.com/zulong210220/lrpc/log"
	"net/http"
)

const (
	headerContentType = "Content-Type"
	ContentType       = "text/plain; charset=utf-8"
)

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != consts.MethodConnect {
		w.Header().Set(headerContentType, ContentType)
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = io.WriteString(w, "405 must CONNECT")
		return
	}

	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		log.Error("", "rpc hijacking ", r.RemoteAddr, " : ", err)
		return
	}

	_, _ = io.WriteString(conn, "HTTP/1.0 "+consts.Connected+"\n\n")
	s.ServeConn(conn)
}

func (s *Server) HandleHTTP() {
	http.Handle(consts.DefaultRpcPath, s)
	http.Handle(consts.DefaultDebugPath, debugHTTP{s})
	log.Info("", "Server.HandleHTTP serveing....")
}

func HandleHTTP() {
	DefaultServer.HandleHTTP()
}
