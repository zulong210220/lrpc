package main

/*
 * Author : lijinya
 * Email : yajin160305@gmail.com
 * File : main.go
 * CreateDate : 2021-06-29 15:13:10
 * */

import (
	"encoding/json"
	"fmt"
	"lrpc/lcode"
	"lrpc/rpc"
	"net"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func testStartServer(addr chan string) {
	fun := "startServer"
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		logrus.Fatalf("%s network error: %v", fun, err)
		return
	}

	logrus.Infof("%s start rpc server on %v", fun, ln.Addr())
	addr <- ln.Addr().String()
	rpc.Accept(ln)
}

func TestCode(t *testing.T) {
	fun := "TestCode"

	lcode.Init()
	addr := make(chan string)
	go testStartServer(addr)

	conn, _ := net.Dial("tcp", <-addr)
	defer func() {
		_ = conn.Close()
	}()

	time.Sleep(time.Second)

	_ = json.NewEncoder(conn).Encode(rpc.DefaultOption)
	cc := lcode.NewGobCodec(conn)

	for i := 0; i < 5; i++ {
		h := &lcode.Header{
			ServiceMethod: "Foo.sum",
			Seq:           uint64(i),
		}

		_ = cc.Write(h, fmt.Sprintf("lrpc req %d", h.Seq))
		_ = cc.ReadHeader(h)

		var reply string
		_ = cc.ReadBody(&reply)
		logrus.Infof("%s reply:%v", fun, reply)
	}
}

/* vim: set tabstop=4 set shiftwidth=4 */
