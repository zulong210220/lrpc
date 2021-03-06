package models

import (
	"math/rand"
	"sort"
	"time"

	proto "github.com/golang/protobuf/proto"
)

type GogoProtoColorGroup struct {
	Id     *int32   `protobuf:"varint,1,req,name=id" json:"id,omitempty"`
	Name   *string  `protobuf:"bytes,2,req,name=name" json:"name,omitempty"`
	Colors []string `protobuf:"bytes,3,rep,name=colors" json:"colors,omitempty"`
}

func (m *GogoProtoColorGroup) Reset()         { *m = GogoProtoColorGroup{} }
func (m *GogoProtoColorGroup) String() string { return proto.CompactTextString(m) }
func (*GogoProtoColorGroup) ProtoMessage()    {}

type GogoProtoColorGroupRsp struct {
	Id *int32 `protobuf:"varint,1,req,name=id" json:"id,omitempty"`
}

func (m *GogoProtoColorGroupRsp) Reset() { *m = GogoProtoColorGroupRsp{} }
func (m *GogoProtoColorGroupRsp) String() string {
	return "RESP"
	//return proto.CompactTextString(m)
}
func (*GogoProtoColorGroupRsp) ProtoMessage() {}

type Gogo struct {
}

func (g *Gogo) Demo(req GogoProtoColorGroup, resp *GogoProtoColorGroupRsp) error {
	if resp == nil {
		resp = &GogoProtoColorGroupRsp{}
	}
	resp.Id = new(int32)
	*resp.Id = (*req.Id) / 3
	k := rand.Intn(500)
	if k < 50 {
		k = 50
	}
	sort.Strings(req.Colors)
	time.Sleep(time.Duration(k) * time.Millisecond)
	return nil
}
