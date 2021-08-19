package lcode

import (
	"bytes"
	"io"
	"sync"
)

type Header struct {
	ServiceMethod string
	Seq           uint64
	Error         string
}

type Codec interface {
	io.Closer
	Read(*Message) error
	Write(*Header, IMessage) error
	Decode([]byte, IMessage) error
}

type IMessage interface {
	Reset()
	String() string
	ProtoMessage()
}

// ---

type NewCodecFunc func(io.ReadWriteCloser) Codec

type Type string

const (
	GobType     Type = "application/gob"
	JsonType    Type = "application/json"
	ProtoType   Type = "application/proto"
	GoProtoType Type = "application/gogoproto"
)

var (
	NewCodecFuncMap map[Type]NewCodecFunc
)

func Init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
	NewCodecFuncMap[JsonType] = NewJsonCodec
	NewCodecFuncMap[ProtoType] = NewProtoCodec
	NewCodecFuncMap[GoProtoType] = NewGoProtoCodec
}

var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func GetBuffer() *bytes.Buffer {
	buffer := bufferPool.Get().(*bytes.Buffer)
	return buffer
}

func PutBuffer(buffer *bytes.Buffer) {
	buffer.Reset()
	bufferPool.Put(buffer)
}

/* vim: set tabstop=4 set shiftwidth=4 */
