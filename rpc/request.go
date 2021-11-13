package rpc

import (
	"fmt"
	"reflect"

	"github.com/zulong210220/lrpc/lcode"
)

type (
	InvalidRequest struct{}
)

func (ir *InvalidRequest) Reset() {

}
func (ir *InvalidRequest) String() string {
	return "invalid request"
}
func (ir *InvalidRequest) ProtoMessage() {

}

var (
	invalidRequest = &InvalidRequest{}
)

type request struct {
	h            *lcode.Header
	argv, replyv reflect.Value
	mType        *methodType
	svc          *service
}

func (r *request) Header() *lcode.Header {
	h := &lcode.Header{
		ServiceMethod: r.h.ServiceMethod,
		Seq:           r.h.Seq,
		Error:         r.h.Error,
	}
	return h
}

func (r *request) String() string {
	if r == nil {
		return "nil"
	}
	return fmt.Sprintf("header:%+v argv:%+v", r.h, r.argv)
}
