package main

/*
 * File : client.go
 * CreateDate : 2021-11-06 13:39:06
 * */

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/zulong210220/lrpc/log"
	"github.com/zulong210220/lrpc/rpc"
)

func PackHeader() []byte {
	fun := "PH"
	dataBuf := bytes.NewBuffer([]byte{})
	var err error

	err = jsoniter.NewEncoder(dataBuf).Encode(rpc.DefaultOption)
	if err != nil {
		log.Errorf("", "%s rpc client options failed err:%v", fun, err)
		return nil
	}

	pkg := dataBuf.Bytes()

	dataBuf = bytes.NewBuffer([]byte{})

	n := uint16(len(pkg))
	err = binary.Write(dataBuf, binary.BigEndian, n)
	if err != nil {
		log.Errorf("PackHeader", " binary.Write len Option failed err:%v", err)
		return nil
	}

	err = binary.Write(dataBuf, binary.BigEndian, pkg)
	if err != nil {
		log.Errorf("PackHeader", " binary.Write Option failed err:%v", err)
		return nil
	}

	return dataBuf.Bytes()
}

func main() {
	fmt.Println("vim-go")
	fun := "main"
	network := "tcp"
	addr := ":8082"

	wg := sync.WaitGroup{}
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			conn, err := net.DialTimeout(network, addr, 5*time.Second)
			if err != nil {
				fmt.Printf("%s DialTimeout failed network:%s addr:%s err:%v", fun, network, addr, err)
				return
			}

			defer func() {
				_ = conn.Close()
			}()

			var n int
			pkg := PackHeader()
			if len(pkg) == 0 {
				fmt.Println("PH failed")
				return
			}
			n, err = conn.Write(pkg)
			if err != nil {
				fmt.Printf("%s rpc client options failed write n:%d err:%v\n", fun, n, err)
				return
			}

			for j := 0; j < 3; j++ {
				buf := []byte(fmt.Sprintf("ping%d\n", i+j))
				conn.Write(buf)
			}
			time.Sleep(1 * time.Second)
		}(i * 100)
	}
	wg.Wait()
}

/* vim: set tabstop=4 set shiftwidth=4 */
