package rpc

import (
	"io"
	"lrpc/log"
	"net/http"
)

const (
	connected        = "200 Connected to lrpc"
	defaultRpcPath   = "/_lrpc_"
	defaultDebugPath = "/debug/_lrpc_"

	methodConnect     = "CONNECT"
	headerContentType = "Content-Type"
	ContentType       = "text/plain; charset=utf-8"
)

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == methodConnect {
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

	_, _ = io.WriteString(conn, "HTTP/1.0 "+connected+"\n\n")
	s.ServeConn(conn)
}

func (s *Server) HandleHTTP() {
	http.Handle(defaultRpcPath, s)
}

func HandleHTTP() {
	DefaultServer.HandleHTTP()
}
