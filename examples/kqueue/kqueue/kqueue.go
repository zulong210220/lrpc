package kqueue

import (
	"fmt"
	"syscall"

	"github.com/zulong210220/lrpc/examples/kqueue/socket"
)

type EventLoop struct {
	KqueueFileDescriptor int
	SocketFileDescriptor int
}

func NewEventLoop(s *socket.Socket) (*EventLoop, error) {
	kQueue, err := syscall.Kqueue()
	if err != nil {
		return nil,
			fmt.Errorf("failed to create kqueue file descriptor (%v)", err)
	}

	changeEvent := syscall.Kevent_t{
		Ident:  uint64(s.FileDescriptor),
		Filter: syscall.EVFILT_READ,
		Flags:  syscall.EV_ADD | syscall.EV_ENABLE,
		Fflags: 0,
		Data:   0,
		Udata:  nil,
	}

	changeEventRegistered, err := syscall.Kevent(
		kQueue,
		[]syscall.Kevent_t{changeEvent},
		nil,
		nil,
	)
	if err != nil || changeEventRegistered == -1 {
		return nil,
			fmt.Errorf("failed to register change event (%v)", err)
	}

	return &EventLoop{
		KqueueFileDescriptor: kQueue,
		SocketFileDescriptor: s.FileDescriptor,
	}, nil
}

type Handler func(*socket.Socket)

func (eventLoop *EventLoop) Handle(handler Handler) {
	for {
		newEvents := make([]syscall.Kevent_t, 10)
		numNewEvents, err := syscall.Kevent(
			eventLoop.KqueueFileDescriptor,
			nil,
			newEvents,
			nil,
		)
		if err != nil {
			continue
		}

		for i := 0; i < numNewEvents; i++ {
			currentEvent := newEvents[i]
			eventFileDescriptor := int(currentEvent.Ident)

			if currentEvent.Flags&syscall.EV_EOF != 0 {
				// client closing connection
				syscall.Close(eventFileDescriptor)
			} else if eventFileDescriptor == eventLoop.SocketFileDescriptor {
				// new incoming connection
				socketConnection, _, err :=
					syscall.Accept(eventFileDescriptor)
				if err != nil {
					continue
				}

				socketEvent := syscall.Kevent_t{
					Ident:  uint64(socketConnection),
					Filter: syscall.EVFILT_READ,
					Flags:  syscall.EV_ADD,
					Fflags: 0,
					Data:   0,
					Udata:  nil,
				}
				socketEventRegistered, err := syscall.Kevent(
					eventLoop.KqueueFileDescriptor,
					[]syscall.Kevent_t{socketEvent},
					nil,
					nil,
				)
				if err != nil || socketEventRegistered == -1 {
					continue
				}

			} else if currentEvent.Filter&syscall.EVFILT_READ != 0 {
				// data available -> forward to handler
				handler(&socket.Socket{
					FileDescriptor: int(eventFileDescriptor),
				})
			}

			// ignore all other events
		}
	}
}
