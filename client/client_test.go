package client

/*
 * Author : lijinya
 * Email : yajin160305@gmail.com
 * File : client_test.go
 * CreateDate : 2021-07-07 16:27:51
 * */

import (
	"lrpc/rpc"
	"net"
	"strings"
	"testing"
	"time"
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

/* vim: set tabstop=4 set shiftwidth=4 */
