package lcode

import "io"

type Header struct {
	ServiceMethod string
	Seq           uint64
	Error         string
}

type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

// ---

type NewCodecFunc func(io.ReadWriteCloser) Codec

type Type string

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json"
)

var (
	NewCodecFuncMap map[Type]NewCodecFunc
)

func Init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}

/* vim: set tabstop=4 set shiftwidth=4 */
