package main

/*
 * Author : zulong210220
 * Email : zulong210220@gmail.com
 * File : main.go
 * CreateDate : 2021-06-29 15:13:10
 * */

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/zulong210220/lrpc/client"
	"github.com/zulong210220/lrpc/consts"
	"github.com/zulong210220/lrpc/lcode"
	"github.com/zulong210220/lrpc/log"
	"github.com/zulong210220/lrpc/registry"
	"github.com/zulong210220/lrpc/rpc"
	"github.com/zulong210220/lrpc/xclient"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func testStartServer(addr chan string) {
	fun := "startServer"
	var f rpc.Foo

	err := rpc.Register(&f)
	if err != nil {
		log.Fatalf("", "%s register failed err:%v", fun, err)
		if err != consts.ErrRegDup {
			return
		}
	}

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("", "%s network error: %v", fun, err)
		return
	}

	log.Infof("", "%s start rpc server on %v", fun, ln.Addr())
	addr <- ln.Addr().String()
	rpc.Accept(ln)
}

func TestCode(t *testing.T) {
	fun := "TestCode"

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
			ServiceMethod: "Foo.Sum",
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
		wg  sync.WaitGroup
		ctx = context.Background()
	)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := fmt.Sprintf("client lllrpc req %d", i)
			var reply string

			err := c.Call(ctx, "Foo.Sum", args, &reply)
			if err != nil {
				log.Errorf("%s call Foo.Sum failed err:%v", fun, err)
				return
			}
			log.Infof("%s reply:%v", fun, reply)
		}(i)
	}
	wg.Wait()
}

func TestReg(t *testing.T) {

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

	var (
		wg  sync.WaitGroup
		ctx = context.Background()
	)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			var reply int
			args := &rpc.Args{Num1: i, Num2: i * i}

			err := c.Call(ctx, "Foo.Sum", args, &reply)
			if err != nil {
				log.Fatalf("", "call Foo.Sum failed i:%d err:%v", i, err)
				return
			}
			log.Infof("", "call success %d + %d = %d", args.Num1, args.Num2, reply)
		}(i)
	}
	wg.Wait()
}

func TestTimeout(t *testing.T) {
	t.Parallel()

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

	ad := <-addr

	t.Run("client timeout", func(t *testing.T) {
		c, _ := client.Dial("tcp", ad)
		ctx, _ := context.WithTimeout(context.Background(), time.Second)
		var reply int
		err := c.Call(ctx, "Foo.Timeout", 1, &reply)
		if !(err != nil && strings.Contains(err.Error(), ctx.Err().Error())) {
			t.Fatal("expected timeout ", err)
		}
		log.Infof("", "client timeout reply:%d err:%v", reply, err)
	})

	t.Run("server timeout", func(t *testing.T) {
		c, _ := client.Dial("tcp", ad, &rpc.Option{
			HandleTimeout: time.Second,
		})
		var reply int
		err := c.Call(context.Background(), "Foo.Timeout", 1, &reply)
		if err == nil {
			t.Fatal("expected timeout  err", err)
		}
		if !strings.Contains(err.Error(), "handle timeout") {
			t.Fatal("expected timeout ", err)
		}
		log.Infof("", "server timeout reply:%d err:%v", reply, err)
	})
}

func testHttpServer(addr chan string) {
	fun := "startServer"
	var f rpc.Foo

	err := rpc.Register(&f)
	if err != nil {
		log.Fatalf("%s register failed err:%v", fun, err)
		return
	}

	ln, err := net.Listen("tcp", ":19999")
	if err != nil {
		log.Fatalf("%s network error: %v", fun, err)
		return
	}

	log.Infof("%s start rpc server on %v", fun, ln.Addr())
	rpc.HandleHTTP()
	addr <- ln.Addr().String()
	http.Serve(ln, nil)
}

// TODO http connect 405
func httpCall(addr chan string) {
	ad := <-addr
	//time.Sleep(10 * time.Second)
	cli, _ := client.DialHTTP("tcp", ad)
	defer func() {
		_ = cli.Close()
	}()

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := &rpc.Args{Num1: i, Num2: i * i}
			var reply int
			if err := cli.Call(context.Background(), "Foo.Sum", args, &reply); err != nil {
				log.Fatal("", "call Foo.Sum error:", err)
			}
			log.Infof("", "%d + %d = %d", args.Num1, args.Num2, reply)
		}(i)
	}
	wg.Wait()
}

func TestHttp(t *testing.T) {
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

	ch := make(chan string)
	go httpCall(ch)
	testHttpServer(ch)
}

func foo(xc *xclient.XClient, ctx context.Context, typ, serviceMethod string, args *rpc.Args) {
	var reply int
	var err error
	switch typ {
	case "call":
		err = xc.Call(ctx, serviceMethod, args, &reply)
	case "broadcast":
		err = xc.Broadcast(ctx, serviceMethod, args, &reply)
	}
	if err != nil {
		log.Errorf("", "%s %s error: %v", typ, serviceMethod, err)
	} else {
		log.Infof("", "%s %s success: %d + %d = %d", typ, serviceMethod, args.Num1, args.Num2, reply)
	}
}

func call(addr1, addr2 string) {
	d := xclient.NewMultiServerDiscovery([]string{"tcp@" + addr1, "tcp@" + addr2})
	xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
	defer func() { _ = xc.Close() }()
	// send request & receive response
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			foo(xc, context.Background(), "call", "Foo.Sum", &rpc.Args{Num1: i, Num2: i * i})
		}(i)
	}
	wg.Wait()
}

func broadcast(addr1, addr2 string) {
	d := xclient.NewMultiServerDiscovery([]string{"tcp@" + addr1, "tcp@" + addr2})
	xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
	defer func() { _ = xc.Close() }()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			foo(xc, context.Background(), "broadcast", "Foo.Sum", &rpc.Args{Num1: i, Num2: i * i})
			// expect 2 - 5 timeout
			ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
			foo(xc, ctx, "broadcast", "Foo.Sleep", &rpc.Args{Num1: i, Num2: i * i})
		}(i)
	}
	wg.Wait()
}

func TestBroadcast(t *testing.T) {
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

	ch1 := make(chan string)
	ch2 := make(chan string)
	// start two servers
	go testStartServer(ch1)
	go testStartServer(ch2)

	addr1 := <-ch1
	addr2 := <-ch2

	call(addr1, addr2)
	broadcast(addr1, addr2)
	time.Sleep(time.Second)
}

func startRegistry(wg *sync.WaitGroup) {
	l, _ := net.Listen("tcp", ":19999")
	registry.HandleHTTP()
	wg.Done()
	_ = http.Serve(l, nil)
}

func startRegServer(registryAddr string, wg *sync.WaitGroup) {
	var foo rpc.Foo
	l, _ := net.Listen("tcp", ":0")
	server := rpc.NewServer()
	_ = server.Register(&foo)
	registry.Heartbeat(registryAddr, "tcp@"+l.Addr().String(), 0)
	wg.Done()
	server.Accept(l)
}

func regCall(registry string) {
	d := xclient.NewRegistryDiscovery(registry, 0)
	xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
	defer func() { _ = xc.Close() }()
	// send request & receive response
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			foo(xc, context.Background(), "call", "Foo.Sum", &rpc.Args{Num1: i, Num2: i * i})
		}(i)
	}
	wg.Wait()
}

func regBroadcast(registry string) {
	d := xclient.NewRegistryDiscovery(registry, 0)
	xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
	defer func() { _ = xc.Close() }()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			foo(xc, context.Background(), "broadcast", "Foo.Sum", &rpc.Args{Num1: i, Num2: i * i})
			// expect 2 - 5 timeout
			ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
			foo(xc, ctx, "broadcast", "Foo.Sleep", &rpc.Args{Num1: i, Num2: i * i})
		}(i)
	}
	wg.Wait()
}

func TestDiscovery(t *testing.T) {
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

	registryAddr := "http://localhost:19999/lrpc/registry"
	var wg sync.WaitGroup
	wg.Add(1)
	go startRegistry(&wg)
	wg.Wait()

	time.Sleep(time.Second)
	wg.Add(2)
	go startRegServer(registryAddr, &wg)
	go startRegServer(registryAddr, &wg)
	wg.Wait()

	time.Sleep(time.Second)
	regCall(registryAddr)
	regBroadcast(registryAddr)
	time.Sleep(time.Second)
}

/* vim: set tabstop=4 set shiftwidth=4 */
