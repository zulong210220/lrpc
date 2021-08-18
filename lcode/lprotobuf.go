package lcode

/*
 * Author : lijinya
 * Email : yajin160305@gmail.com
 * File : lgogoproto.go
 * CreateDate : 2021-08-18 16:13:51
 * */

import (
	"bytes"
	"encoding/binary"
	"io"

	goproto "github.com/gogo/protobuf/proto"
	"github.com/zulong210220/lrpc/log"
)

type GoProtoCodec struct {
	conn io.ReadWriteCloser
}

func NewGoProtoCodec(conn io.ReadWriteCloser) *GoProtoCodec {
	return &GoProtoCodec{
		conn: conn,
	}
}

// ---

func (jc *GoProtoCodec) Read(msg *Message) error {
	fun := "GoProtoCodec.Read"
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

func (jc *GoProtoCodec) Decode(data []byte, body goproto.Message) error {
	return goproto.Unmarshal(data, body)
}

func (pc *GoProtoCodec) Write(h *Header, body goproto.Message) (err error) {
	fun := "GoProtoCodec.Write"
	defer func() {
		if err != nil {
			_ = pc.Close()
		}
	}()

	var bs []byte
	bs, err = goproto.Marshal(body)
	if err != nil {
		log.Errorf("JEM", "%s rpc codec: goproto Marshal failed error :%v", fun, err)
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
	n, err = pc.conn.Write(tbs)
	if err != nil {
		log.Errorf("JC", "%s rpc codec: goproto error write : %d total :%v", fun, n, err)
		return
	}

	n, err = pc.conn.Write(bs)
	if err != nil {
		log.Errorf("JC", "%s rpc codec: goproto error write : %d buffer :%v", fun, n, err)
	}

	return
}

func (pc *GoProtoCodec) Close() error {
	return pc.conn.Close()
}

/* vim: set tabstop=4 set shiftwidth=4 */
