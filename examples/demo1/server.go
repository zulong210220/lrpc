package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/zulong210220/lrpc/lcode"
	"github.com/zulong210220/lrpc/log"
	"github.com/zulong210220/lrpc/rpc"
)

func main() {
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

	s := rpc.NewServer()
	s.Init(&rpc.Config{
		//EtcdAddr:    []string{"127.0.0.1:2379"},
		EtcdAddr:    []string{"127.0.0.1:4001", "127.0.0.1:5001", "127.0.0.1:6001"},
		EtcdTimeout: 5,
		ServerName:  "demo1",
	})

	var f rpc.Foo

	s.Register(&f)
	go s.Run()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGALRM, syscall.SIGTERM, syscall.SIGUSR1)
	<-c

	s.Stop()
}
