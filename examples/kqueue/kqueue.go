package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/zulong210220/lrpc/consts"
	"github.com/zulong210220/lrpc/log"
	"github.com/zulong210220/lrpc/rpc"

	"github.com/zulong210220/lrpc/examples/kqueue/kqueue"
	"github.com/zulong210220/lrpc/examples/kqueue/socket"
)

// test
// curl http://127.0.0.1:8082

// 来源
// https://dev.to/frosnerd/writing-a-simple-tcp-server-using-kqueue-cah

// github library
// https://github.com/fsnotify/fsnotify
// https://github.com/mailru/easygo

func main() {
	s, err := socket.Listen("127.0.0.1", 8082)
	if err != nil {
		fmt.Println("Failed to create Socket:", err)
		os.Exit(1)
	}

	eventLoop, err := kqueue.NewEventLoop(s)
	if err != nil {
		fmt.Println("Failed to create event loop:", err)
		os.Exit(1)
	}

	fmt.Println("Server started. Waiting for incoming connections. ^C to exit.")

	eventLoop.Handle(func(s *socket.Socket) {
		var opt rpc.Option
		//data, err := ioutil.ReadAll(conn)

		var data = make([]byte, consts.HandleshakeBufLen)
		numBytesRead, err := syscall.Read(s.FileDescriptor, data)
		if err != nil {
			numBytesRead = 0
			return
		}
		fmt.Println("Accept", numBytesRead)

		fun := "Accept"
		if err != nil {
			log.Errorf("", "%s ioutil.ReadAll options error:%v", fun, err)
			return
		}
		//err := json.NewDecoder(conn).Decode(&opt)
		n := 0
		for n < len(data) {
			if data[n] == '}' {
				break
			}
			n++
		}
		data = data[:n+1]

		err = json.Unmarshal(data, &opt)
		if err != nil {
			log.Errorf("", "%s rpc server options error:%v", fun, err)
			return
		}

		if opt.MagicNumber != rpc.MagicNumber {
			log.Errorf("", "%s rpc server invalid magic number %x", fun, opt.MagicNumber)
			return
		}
		fmt.Println("Opt", opt)

		reader := bufio.NewReader(s)
		for {
			line, err := reader.ReadString('\n')
			if err != nil || strings.TrimSpace(line) == "" {
				break
			}
			fmt.Println(line)
			s.Write([]byte(line))
			s.Write([]byte("--------"))
		}
		s.Close()
	})
}
