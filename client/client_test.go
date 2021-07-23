package client

/*
 * Author : lijinya
 * Email : yajin160305@gmail.com
 * File : client_test.go
 * CreateDate : 2021-07-07 16:27:51
 * */

import (
	"net"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/zulong210220/lrpc/rpc"
)

func TestTimeout(t *testing.T) {
	t.Parallel()

	ln, _ := net.Listen("tcp", ":0")

	f := func(conn net.Conn, opt *rpc.Option) (*Client, error) {
		_ = conn.Close()
		time.Sleep(2 * time.Second)
		return nil, nil
	}

	t.Run("timeout", func(t *testing.T) {
		_, err := dialTimeout(f, "tcp", ln.Addr().String(), &rpc.Option{
			ConnectTimeout: time.Second,
		})

		if !(err != nil && strings.Contains(err.Error(), "connect timeout")) {
			t.Fatal("expect timeout")
		}
	})

	t.Run("timeout", func(t *testing.T) {
		_, err := dialTimeout(f, "tcp", ln.Addr().String(), &rpc.Option{
			ConnectTimeout: 0,
		})

		if err != nil {
			t.Fatal("unexpected timeout should no limit", err)
		}
	})
}

func TestXDial(t *testing.T) {
	if runtime.GOOS == "linux" {
		ch := make(chan struct{})
		addr := "/tmp/lrpc.sock"
		go func() {
			_ = os.Remove(addr)
			l, err := net.Listen("unix", addr)
			if err != nil {
				t.Fatal("failed to listen unix sock")
			}
			ch <- struct{}{}
			rpc.Accept(l)
		}()
		<-ch
		_, err := XDial("unix@" + addr)
		if err != nil {
			t.Fatal("failed to connect unix socket")
		}
	}
}

/* vim: set tabstop=4 set shiftwidth=4 */
