package socket

import (
	"fmt"
	"net"
	"strconv"
	"syscall"
)

type Socket struct {
	FileDescriptor int
}

func (socket Socket) Read(bytes []byte) (int, error) {
	if len(bytes) == 0 {
		return 0, nil
	}
	numBytesRead, err := syscall.Read(socket.FileDescriptor, bytes)
	if err != nil {
		numBytesRead = 0
	}
	return numBytesRead, err
}

func (socket Socket) Write(bytes []byte) (int, error) {
	numBytesWritten, err :=
		syscall.Write(socket.FileDescriptor, bytes)
	if err != nil {
		numBytesWritten = 0
	}
	return numBytesWritten, err
}

func (socket *Socket) Close() error {
	return syscall.Close(socket.FileDescriptor)
}

func (socket *Socket) String() string {
	return strconv.Itoa(socket.FileDescriptor)
}

func Listen(ip string, port int) (*Socket, error) {
	socket := &Socket{}

	socketFileDescriptor, err :=
		syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to create socket (%v)", err)
	}
	socket.FileDescriptor = socketFileDescriptor

	socketAddress := &syscall.SockaddrInet4{Port: port}
	copy(socketAddress.Addr[:], net.ParseIP(ip))
	if err = syscall.Bind(socket.FileDescriptor, socketAddress); err != nil {
		return nil, fmt.Errorf("failed to bind socket (%v)", err)
	}

	if err = syscall.Listen(socket.FileDescriptor, syscall.SOMAXCONN); err != nil {
		return nil, fmt.Errorf("failed to listen on socket (%v)", err)
	}

	return socket, nil
}
