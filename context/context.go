package context

/*
 * Author : lijinya
 * Email : yajin160305@gmail.com
 * File : context.go
 * CreateDate : 2021-08-23 10:51:17
 * */

import (
	"context"
	"fmt"
	"reflect"
)

type Context struct {
	meta map[interface{}]interface{}
	context.Context
}

func NewContext(ctx context.Context) *Context {
	return &Context{
		Context: ctx,
	}
}

func (c *Context) Value(key interface{}) interface{} {
	if c.meta == nil {
		c.meta = make(map[interface{}]interface{})
	}

	if v, ok := c.meta[key]; ok {
		return v
	}
	return c.Context.Value(key)
}

func (c *Context) SetValue(key, val interface{}) {
	if c.meta == nil {
		c.meta = make(map[interface{}]interface{})
	}
	c.meta[key] = val
}

func (c *Context) String() string {
	return fmt.Sprintf("%v.WithValue(%v)", c.Context, c.meta)
}

func WithValue(parent context.Context, key, val interface{}) *Context {
	if key == nil {
		panic("nil key")
	}
	if !reflect.TypeOf(key).Comparable() {
		panic("key is not comparable")
	}

	meta := make(map[interface{}]interface{})
	meta[key] = val
	return &Context{Context: parent, meta: meta}
}

func WithLocalValue(ctx *Context, key, val interface{}) *Context {
	if key == nil {
		panic("nil key")
	}
	if !reflect.TypeOf(key).Comparable() {
		panic("key is not comparable")
	}

	if ctx.meta == nil {
		ctx.meta = make(map[interface{}]interface{})
	}

	ctx.meta[key] = val
	return ctx
}

/* vim: set tabstop=4 set shiftwidth=4 */
