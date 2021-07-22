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
	"github.com/zulong210220/lrpc/client"
	"github.com/zulong210220/lrpc/rpc"
	"reflect"
	"sync"
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

func (xc *XClient) call(rpcAddr string, ctx context.Context, sm string, args, reply interface{}) error {
	cli, err := xc.dial(rpcAddr)
	if err != nil {
		return err
	}
	return cli.Call(ctx, sm, args, reply)
}

func (xc *XClient) Call(ctx context.Context, sm string, args, reply interface{}) error {
	rpcAddr, err := xc.d.Get(xc.mode)
	if err != nil {
		return err
	}

	return xc.call(rpcAddr, ctx, sm, args, reply)
}

func (xc *XClient) Broadcast(ctx context.Context, sm string, args, reply interface{}) error {
	ss, err := xc.d.GetAll()
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
				clonedReply interface{}
			)
			if reply != nil {
				clonedReply = reflect.New(reflect.ValueOf(reply).Elem().Type()).Interface()
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
