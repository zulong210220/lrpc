package registry

/*
 * Author : lijinya
 * Email : yajin160305@gmail.com
 * File : registry.go
 * CreateDate : 2021-07-21 18:42:21
 * */

import (
	"lrpc/consts"
	"lrpc/log"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type ServerItem struct {
	Addr  string
	start time.Time
}

type Registry struct {
	timeout time.Duration
	mu      sync.Mutex
	servers map[string]*ServerItem
}

func New(to time.Duration) *Registry {
	return &Registry{
		servers: make(map[string]*ServerItem),
		timeout: to,
	}
}

var (
	DefaultRegister = New(consts.DefaultRegTimeout)
)

func (r *Registry) putServer(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	s := r.servers[addr]
	if s == nil {
		r.servers[addr] = &ServerItem{
			Addr:  addr,
			start: time.Now(),
		}
	} else {
		s.start = time.Now()
	}
}

func (r *Registry) aliveServers() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var (
		alive []string
	)

	for addr, s := range r.servers {
		if r.timeout == 0 || s.start.Add(r.timeout).After(time.Now()) {
			alive = append(alive, addr)
		} else {
			delete(r.servers, addr)
		}
	}

	sort.Strings(alive)
	return alive
}

func (r *Registry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		w.Header().Set("X-lrpc-servers", strings.Join(r.aliveServers(), ","))
	case "POST":
		addr := req.Header.Get("X-lrpc-server")
		if addr == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		r.putServer(addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (r *Registry) HandleHTTP(rp string) {
	http.Handle(rp, r)
	log.Info("", "rpc registry path:", rp)
}

func HandleHTTP() {
	DefaultRegister.HandleHTTP(consts.DefaultRegPath)
}

func Heartbeat(reg, addr string, dur time.Duration) {
	if dur == 0 {
		dur = consts.DefaultRegTimeout
	}

	go func(dur time.Duration) {
		var (
			err error
		)
		t := time.NewTicker(dur)
		for err == nil {
			err = sendHeartbeat(reg, addr)
			<-t.C
		}
	}(dur)
}

var (
	client = &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
)

func sendHeartbeat(reg, addr string) error {
	fun := "sendHeartbeat"
	log.Infof("", "%s send heart beat to registry:%s", addr, reg)

	req, _ := http.NewRequest("POST", reg, nil)
	req.Header.Set("X-lrpc-server", addr)
	_, err := client.Do(req)
	if err != nil {
		log.Errorf("", "%s rpc server: heart beat err:%v", fun, err)
		return err
	}
	return nil
}

/* vim: set tabstop=4 set shiftwidth=4 */
