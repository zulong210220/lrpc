package main

import (
	"io/ioutil"
	"log"
	"os"
	"syscall"
)

const BufferSize = 1024

//完全以io复用的方式从标准输入流读数据，将数据输出到标准输出流中
func main2() {
	var changes []syscall.Kevent_t
	events := make([]syscall.Kevent_t, 128)
	kq, err := syscall.Kqueue() //创建kqueue
	if err != nil {
		panic(err)
	}
	stdinFd := syscall.Stdin
	stdoutFd := syscall.Stdout //设备从标准输入和标准输出流中读入数据
	changes = append(changes, syscall.Kevent_t{Ident: uint64(stdinFd), Filter: syscall.EVFILT_READ, Flags: syscall.EV_ADD})
	changes = append(changes, syscall.Kevent_t{Ident: uint64(stdoutFd), Filter: syscall.EVFILT_WRITE, Flags: syscall.EV_ADD})
	//设置读写事件并加入到监听事件列表中
	var nev, nread, nwrite int //准备就绪的事件数、已读字节数、已写字节数
	var buffer []byte
	for {
		nev, err = syscall.Kevent(kq, changes, events, nil)
		if err != nil && err != syscall.EINTR {
			log.Fatal(err)
		}
		for i := 0; i < nev; i++ {
			event := events[i]
			ev_fd := int(event.Ident)
			if err := syscall.SetNonblock(ev_fd, true); err != nil {
				//设置为非阻塞模式，保证输入输出缓冲区有数据就绪就写入写出，避免阻塞影响性能
				panic(err)
			}
			if ev_fd == syscall.Stdin && nread < BufferSize { //输入流就绪且缓冲区还有数据继续读
				buffer, err = ioutil.ReadAll(os.Stdin) //存疑，不知道为什么直接调用systm.Read无法从标准输入读取数据
				//if err != nil {
				//	panic(err)
				//}
				if len(buffer) == 0 {
					panic("no data read!")
				}
				nread += len(buffer)
			}
			if ev_fd == syscall.Stdout && nread > 0 { //输出流就绪且缓冲区还能继续写入
				nwrite, err = syscall.Write(ev_fd, buffer)
				if err != nil {
					panic(err)
				}
				if nwrite <= 0 {
					panic("no data write")
				}
				buffer = buffer[nwrite:]
				nread -= nwrite
			}
		}
	}

}
