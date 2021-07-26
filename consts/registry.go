package consts

/*
 * Author : lijinya
 * Email : yajin160305@gmail.com
 * File : registry.go
 * CreateDate : 2021-07-21 18:44:24
 * */

import "time"

const (
	DefaultRegPath       = "/lrpc/registry"
	DefaultRegTimeout    = time.Minute
	DefaultUpdateTimeout = 10 * time.Second
	DefaultRegLease      = 5
)

/* vim: set tabstop=4 set shiftwidth=4 */
