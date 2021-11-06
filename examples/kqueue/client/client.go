package main

/*
 * File : client.go
 * CreateDate : 2021-11-06 13:39:06
 * */

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/zulong210220/lrpc/consts"
	"github.com/zulong210220/lrpc/log"
	"github.com/zulong210220/lrpc/rpc"
)

func main() {
	fmt.Println("vim-go")
	fun := "main"
	network := "tcp"
	addr := ":8082"

	conn, err := net.DialTimeout(network, addr, 5*time.Second)
	if err != nil {
		log.Errorf("", "%s DialTimeout failed network:%s addr:%s err:%v", fun, network, addr, err)
		return
	}

	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()

	data, err := json.Marshal(rpc.DefaultOption)
	if err != nil {
		log.Errorf("", "%s rpc client options failed err:%v", fun, err)
		_ = conn.Close()
		return
	}

	buf := make([]byte, consts.HandleshakeBufLen)
	var n int
	copy(buf, data)
	n, err = conn.Write(buf)
	if err != nil {
		log.Errorf("", "%s rpc client options failed write n:%d err:%v", fun, n, err)
		_ = conn.Close()
		return
	}

	for i := 0; i < 10; i++ {
		buf := []byte(fmt.Sprintf("ping%d\n", i))
		conn.Write(buf)
	}

}

/* vim: set tabstop=4 set shiftwidth=4 */
