package lcode

import (
	"bytes"
	"sync"

	"github.com/zulong210220/lrpc/consts"
	"github.com/zulong210220/lrpc/utils"
)

type Header struct {
	ServiceMethod string
	Seq           uint64
	TraceId       string
	Error         string
}


type IMessage interface {
	Reset()
	String() string
	ProtoMessage()
}

// ---


type Type string

const (
	GobType     Type = "application/gob"
	JsonType    Type = "application/json"
	ProtoType   Type = "application/proto"
	GoProtoType Type = "application/gogoproto"
)

var (
	NewCodecFuncMap map[Type]bool
	limitedPool     *utils.LimitedPool
)

func Init() {
	NewCodecFuncMap = make(map[Type]bool)
	NewCodecFuncMap[GobType] = true
	NewCodecFuncMap[JsonType] = true
	NewCodecFuncMap[ProtoType] = true
	NewCodecFuncMap[GoProtoType] = true

	limitedPool = utils.NewLimitedPool(consts.BufferPoolSizeMin, consts.BufferPoolSizeMax)
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
