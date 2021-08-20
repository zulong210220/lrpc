package main

import (
	"bufio"
	"log"
	"os"
	"strings"

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
		log.Println("Failed to create Socket:", err)
		os.Exit(1)
	}

	eventLoop, err := kqueue.NewEventLoop(s)
	if err != nil {
		log.Println("Failed to create event loop:", err)
		os.Exit(1)
	}

	log.Println("Server started. Waiting for incoming connections. ^C to exit.")

	eventLoop.Handle(func(s *socket.Socket) {
		reader := bufio.NewReader(s)
		for {
			line, err := reader.ReadString('\n')
			if err != nil || strings.TrimSpace(line) == "" {
				break
			}
			s.Write([]byte(line))
			s.Write([]byte("--------"))
		}
		s.Close()
	})
}
