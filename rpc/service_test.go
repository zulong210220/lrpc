package rpc

/*
 * Author : lijinya
 * Email : yajin160305@gmail.com
 * File : service_test.go
 * CreateDate : 2021-07-06 17:32:20
 * */

import (
	"github.com/zulong210220/lrpc/log"
	"reflect"
	"testing"
)

func TestNS(t *testing.T) {
	var f Foo
	s := newService(&f)

	if len(s.method) != 1 {
		t.Fatalf("wrong service methhod expect 1, but got %d", len(s.method))
	}

	mType := s.method["Sum"]
	if mType == nil {
		t.Fatal("wrong method, Sum should't nil")
	}
	log.Info("", mType)
}

func TestFoo(t *testing.T) {
	var f Foo
	s := newService(&f)
	mType := s.method["Sum"]

	argv := mType.newArgv()
	replyv := mType.newReplyv()

	argv.Set(reflect.ValueOf(Args{Num1: 11, Num2: 22}))
	err := s.call(mType, argv, replyv)

	if !(err == nil && *replyv.Interface().(*int) == 33 && mType.NumCalls() == 1) {
		t.Fatalf("failed to call Foo.Sum")
	}

	argv.Set(reflect.ValueOf(Args{Num1: 11, Num2: 12}))
	err = s.call(mType, argv, replyv)

	if !(err == nil && *replyv.Interface().(*int) == 23 && mType.NumCalls() == 2) {
		t.Fatalf("failed to call Foo.Sum")
	}
}

/* vim: set tabstop=4 set shiftwidth=4 */
