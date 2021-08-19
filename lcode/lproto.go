package lcode

/*
 * Author : lijinya
 * Email : yajin160305@gmail.com
 * File : lproto.go
 * CreateDate : 2021-08-18 16:46:55
 * */

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/golang/protobuf/proto"
	"github.com/zulong210220/lrpc/log"
)

type ProtoCodec struct {
	conn io.ReadWriteCloser
}

func NewProtoCodec(conn io.ReadWriteCloser) Codec {
	return &ProtoCodec{
		conn: conn,
	}
}

func (jc *ProtoCodec) Read(msg *Message) error {
	fun := "ProtoCodec.Read"
	// TODO fix
	var data = make([]byte, 4)
	n, err := jc.conn.Read(data)
	if err != nil {
		log.Errorf("JCR", "%s connection total n:%d failed err:%v", fun, n, err)
		if err == io.EOF {
			// TODO
			return err
		}
		return err
	}

	total := binary.BigEndian.Uint32(data)

	data = make([]byte, total)
	n, err = jc.conn.Read(data)
	if err != nil {
		log.Errorf("JCR", "%s connection data n:%d failed err:%v", fun, n, err)
		if err == io.EOF {
			// TODO
			return err
		}
	}

	err = msg.Unpack(data)

	return err
}

func (jc *ProtoCodec) Decode(data []byte, body IMessage) error {
	return proto.Unmarshal(data, body)
}

func (jc *ProtoCodec) Write(h *Header, body IMessage) (err error) {
	fun := "ProtoCodec.Write"
	defer func() {
		if err != nil {
			_ = jc.Close()
		}
	}()

	var bs []byte
	bs, err = proto.Marshal(body)
	if err != nil {
		log.Errorf("JEM", "%s rpc codec: proto Marshal failed error :%v", fun, err)
		return
	}

	var n int
	msg := &Message{}
	msg.H = h
	msg.B = bs

	bs, err = msg.Pack()
	if err != nil {
		return
	}

	dataBuf := bytes.NewBuffer([]byte{})
	err = binary.Write(dataBuf, binary.BigEndian, uint32(len(bs)))
	if err != nil {
		log.Errorf("JC", "%s binary Write len buffer:%v", fun, err)
		return
	}

	tbs := dataBuf.Bytes()
	n, err = jc.conn.Write(tbs)
	if err != nil {
		log.Errorf("JC", "%s rpc codec: proto error write : %d total :%v", fun, n, err)
		return
	}

	n, err = jc.conn.Write(bs)
	if err != nil {
		log.Errorf("JC", "%s rpc codec: proto error write : %d buffer :%v", fun, n, err)
	}

	return
}

func (gc *ProtoCodec) Close() error {
	return gc.conn.Close()
}

/* vim: set tabstop=4 set shiftwidth=4 */
