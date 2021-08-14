package main

import (
	"context"
	"fmt"
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

	d := xclient.NewEtcdDiscovery(
		//[]string{"127.0.0.1:2379"},
		[]string{"127.0.0.1:4001", "127.0.0.1:5001", "127.0.0.1:6001"},
		1, []string{sn})
	xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
	defer func() { _ = xc.Close() }()

	//TODO 二次连接panic
	time.Sleep(1000 * time.Millisecond)
	for i := 1; i < 9; i++ {
		var reply int
		var err error
		ctx := context.Background()
		//ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		fmt.Println("bef call")
		err = xc.Call(ctx, sn, "Foo.Sum", &rpc.Args{Num1: i, Num2: i * i}, &reply)
		fmt.Println("after call")
		if err != nil {
			log.Errorf("", "Call:%d failed err:%v", i, err)
		}
		log.Infof("", "Call [%d] reply:%d", i, reply)
	}
	log.Infof("", "Call over...")
	fmt.Println("all over......")

	// sleep wait log flush
	//time.Sleep(1000 * time.Millisecond)
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGALRM, syscall.SIGTERM, syscall.SIGUSR1)
	<-c
}
