package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zulong210220/lrpc/lcode"
	"github.com/zulong210220/lrpc/log"
	"github.com/zulong210220/lrpc/rpc"
	"github.com/zulong210220/lrpc/xclient"
)

func main() {
	lcode.Init()

	log.Init(&log.Config{
		Dir:      "./logs",
		FileSize: 256,
		FileNum:  256,
		Env:      "test",
		Level:    "INFO",
		FileName: "client",
	})
	defer log.ForceFlush()

	sn := "demo1"

	d := xclient.NewEtcdDiscovery([]string{"127.0.0.1:2379"}, 1, []string{sn})
	xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
	defer func() { _ = xc.Close() }()

	//TODO 二次连接panic
	time.Sleep(1000 * time.Millisecond)
	for i := 1; i < 9999; i++ {
		var reply int
		var err error
		ctx := context.Background()
		err = xc.Call(ctx, sn, "Foo.Sum", &rpc.Args{Num1: i, Num2: i * i}, &reply)
		if err != nil {
			log.Error("", "Call failed err:", err)
		}
		log.Infof("", "Call reply:%d", reply)
	}

	// sleep wait log flush
	//time.Sleep(1000 * time.Millisecond)
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGALRM, syscall.SIGTERM, syscall.SIGUSR1)
	<-c
}
