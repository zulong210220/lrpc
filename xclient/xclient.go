package xclient

/*
 * Author : lijinya
 * Email : yajin160305@gmail.com
 * File : xclient.go
 * CreateDate : 2021-07-20 18:39:48
 * */

import (
	"context"
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/zulong210220/lrpc/lcode"

	"github.com/zulong210220/lrpc/log"

	"github.com/zulong210220/lrpc/client"
	"github.com/zulong210220/lrpc/rpc"
)

type XClient struct {
	d       Discovery
	mode    SelectMode
	opt     *rpc.Option
	mu      sync.Mutex
	clients map[string]*client.Client
}

var (
	_ io.Closer = (*XClient)(nil)
)

func NewXClient(d Discovery, mode SelectMode, opt *rpc.Option) *XClient {
	return &XClient{
		d:       d,
		mode:    mode,
		opt:     opt,
		clients: make(map[string]*client.Client),
	}
}

func (xc *XClient) Close() error {
	xc.mu.Lock()
	defer xc.mu.Unlock()

	for key, client := range xc.clients {
		_ = client.Close()
		delete(xc.clients, key)
	}
	return nil
}

func (xc *XClient) dial(rpcAddr string) (*client.Client, error) {
	xc.mu.Lock()
	defer xc.mu.Unlock()

	cli, ok := xc.clients[rpcAddr]

	if cli == nil {
		var err error
		cli, err = client.XDial(rpcAddr, xc.opt)
		if err != nil {
			return nil, err
		}
		xc.clients[rpcAddr] = cli
	}

	if ok && !cli.IsAvailable() {
		_ = cli.Close()
		delete(xc.clients, rpcAddr)
		cli = nil
	}

	return cli, nil

}

func (xc *XClient) call(rpcAddr string, ctx context.Context, sm string, args, reply lcode.IMessage) error {
	cli, err := xc.dial(rpcAddr)
	if err != nil {
		log.Errorf("xc call", "XClient.call rpcAddr:%s failed err:%v", rpcAddr, err)
		return err
	}

	begin := time.Now().UnixNano()
	err = cli.Call(ctx, sm, args, reply)
	end := time.Now().UnixNano()

	xc.Observe(rpcAddr, end-begin)
	return err
}

func (xc *XClient) Observe(rpcAddr string, dur int64) {
	xc.d.Observe(rpcAddr, dur)
}

// TODO server close retry
func (xc *XClient) Call(ctx context.Context, sn, sm string, args, reply lcode.IMessage) error {
	rpcAddr, err := xc.d.Get(sn, xc.mode)
	if err != nil {
		log.Errorf("", "XClient.Call Get service:%s mode:%d method:%s failed err:%v", sn, xc.mode, sm, err)
		return err
	}

	rpcAddr = "tcp@" + rpcAddr

	return xc.call(rpcAddr, ctx, sm, args, reply)
}

func (xc *XClient) Broadcast(ctx context.Context, sn, sm string, args, reply lcode.IMessage) error {
	ss, err := xc.d.GetAll(sn)
	if err != nil {
		return err
	}

	var (
		wg sync.WaitGroup
		mu sync.Mutex
		e  error
	)

	replyDone := reply == nil
	ctx, cancel := context.WithCancel(ctx)

	for _, rpcAddr := range ss {
		wg.Add(1)
		go func(rpcAddr string) {
			defer wg.Done()
			var (
				clonedReply lcode.IMessage
			)
			if reply != nil {
				clonedReply = reflect.New(reflect.ValueOf(reply).Elem().Type()).Interface().(lcode.IMessage)
			}
			err := xc.call(rpcAddr, ctx, sm, args, clonedReply)
			mu.Lock()
			defer mu.Unlock()

			if err != nil && e == nil {
				e = err
				cancel()
			}
			if err == nil && !replyDone {
				reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(clonedReply).Elem())
				replyDone = true
			}
		}(rpcAddr)
	}
	wg.Wait()
	return e
}

/* vim: set tabstop=4 set shiftwidth=4 */
