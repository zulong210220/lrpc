package main

/*
 * Author : zulong210220
 * Email : zulong210220@gmail.com
 * File : main.go
 * CreateDate : 2021-06-29 15:13:10
 * */

import (
	"encoding/json"
	"fmt"
	"lrpc/client"
	"lrpc/lcode"
	"lrpc/log"
	"lrpc/rpc"
	"net"
	"sync"
	"testing"
	"time"
)

func testStartServer(addr chan string) {
	fun := "startServer"
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("%s network error: %v", fun, err)
		return
	}

	log.Infof("%s start rpc server on %v", fun, ln.Addr())
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
		log.Infof("%s reply:%v", fun, reply)
	}
}

func TestClient(t *testing.T) {
	fun := "TestClient"

	addr := make(chan string)

	lcode.Init()
	log.Init(&log.Config{
		Dir:      "./logs",
		FileSize: 256,
		FileNum:  256,
		Env:      "test",
		Level:    "INFO",
		FileName: "lrpc",
	})
	defer log.ForceFlush()
	go testStartServer(addr)

	c, _ := client.Dial("tcp", <-addr, rpc.DefaultOption)
	defer func() {
		_ = c.Close()
	}()

	time.Sleep(time.Second)

	var (
		wg sync.WaitGroup
	)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := fmt.Sprintf("client lllrpc req %d", i)
			var reply string

			err := c.Call("Foo.Sum", args, &reply)
			if err != nil {
				log.Errorf("%s call Foo.Sum failed err:%v", fun, err)
				return
			}
			log.Infof("%s reply:%v", fun, reply)
		}(i)
	}
	wg.Wait()
}

/* vim: set tabstop=4 set shiftwidth=4 */
