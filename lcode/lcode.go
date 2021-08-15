package lcode

import "io"

type Header struct {
	ServiceMethod string
	Seq           uint64
	Error         string
}

type Message struct {
	H *Header
	B []byte
}

type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Read(*Message) error
	Write(*Header, interface{}) error
	Decode([]byte, interface{}) error
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
	NewCodecFuncMap[JsonType] = NewJsonCodec
}

/* vim: set tabstop=4 set shiftwidth=4 */
