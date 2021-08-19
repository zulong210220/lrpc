package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zulong210220/lrpc/rpc"

	"github.com/zulong210220/lrpc/models"

	"github.com/zulong210220/lrpc/lcode"
	"github.com/zulong210220/lrpc/log"
	"github.com/zulong210220/lrpc/xclient"

	"github.com/golang/protobuf/proto"
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
		[]string{"127.0.0.1:2379"},
		//[]string{"127.0.0.1:4001", "127.0.0.1:5001", "127.0.0.1:6001"},
		1, []string{sn})
	xc := xclient.NewXClient(d, xclient.RoundRobinSelect, &rpc.Option{
		MagicNumber:    rpc.MagicNumber,
		CodecType:      lcode.JsonType,
		ConnectTimeout: 3 * time.Second,
	})
	defer func() { _ = xc.Close() }()

	//TODO 二次连接panic
	time.Sleep(1000 * time.Millisecond)
	now := time.Now()
	for i := 1; i < 9999; i++ {
		var reply models.GogoProtoColorGroupRsp
		var err error
		ctx := context.Background()
		//ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

		gogoProtobufGroup := models.GogoProtoColorGroup{
			Id:     proto.Int32(int32(i)),
			Name:   proto.String("Reds"),
			Colors: []string{"Crimson", "Red", "Ruby", "Maroon"},
		}

		err = xc.Call(ctx, sn, "Gogo.Demo", &gogoProtobufGroup, &reply)
		if err != nil {
			log.Errorf("", "Call:%d failed err:%v", i, err)
		}
		//if reply.Num != i+i*i {
		log.Infof("", "Call [%d] reply:%d", i, *reply.Id)
		//}

	}
	log.Info("", "Call over...", time.Since(now))
	fmt.Println("all over......", time.Since(now))

	// sleep wait log flush
	//time.Sleep(1000 * time.Millisecond)
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGALRM, syscall.SIGTERM, syscall.SIGUSR1)
	<-c
}
