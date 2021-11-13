package rpc

import "github.com/zulong210220/lrpc/lcode"

type response struct {
	h    *lcode.Header
	body interface{}
}
