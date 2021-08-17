package consts

import "errors"

var (
	ErrRegDup error = errors.New("register dup")
)

const (
	HandleshakeBufLen = 256
)
