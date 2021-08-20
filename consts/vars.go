package consts

import "errors"

var (
	ErrRegDup error = errors.New("register dup")
)

const (
	HandleshakeBufLen = 256

	BufferPoolSizeMin = 4
	BufferPoolSizeMax = 32 * 1024
)
