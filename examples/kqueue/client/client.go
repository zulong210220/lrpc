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

	conn, err := net.DialTimeout(network, addr, 5*time.Second)
	if err != nil {
		log.Errorf("", "%s DialTimeout failed network:%s addr:%s err:%v", fun, network, addr, err)
		return
	}

	defer func() {
		if err != nil {
		}
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
