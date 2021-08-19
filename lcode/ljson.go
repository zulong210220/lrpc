package lcode

import (
	"bytes"
	"encoding/binary"
	"io"

	jsoniter "github.com/json-iterator/go"

	"github.com/zulong210220/lrpc/log"
)

type JsonCodec struct {
	conn io.ReadWriteCloser
}

var (
	_ Codec = (*JsonCodec)(nil)
)

func NewJsonCodec(conn io.ReadWriteCloser) Codec {

	return &JsonCodec{
		conn: conn,
	}
}

// ---

/*
	全部读取了数据
	' {"ServiceMethod":"Foo.Sum","Seq":1,"Error":""}
	{"Num1":1,"Num2":1}
	 '
*/

func (jc *JsonCodec) Read(msg *Message) error {
	fun := "JsonCodec.Read"
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

func (jc *JsonCodec) Decode(data []byte, body IMessage) error {
	return jsoniter.Unmarshal(data, body)
}

func (jc *JsonCodec) Write(h *Header, body IMessage) (err error) {
	fun := "JsonCodec.Write"
	defer func() {
		if err != nil {
			_ = jc.Close()
		}
	}()

	var bs []byte
	bs, err = jsoniter.Marshal(body)
	if err != nil {
		log.Errorf("JEM", "%s rpc codec: json Marshal failed error :%v", fun, err)
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
		log.Errorf("JC", "%s rpc codec: json error write : %d total :%v", fun, n, err)
		return
	}

	n, err = jc.conn.Write(bs)
	if err != nil {
		log.Errorf("JC", "%s rpc codec: json error write : %d buffer :%v", fun, n, err)
	}

	return
}

func (gc *JsonCodec) Close() error {
	return gc.conn.Close()
}

/* vim: set tabstop=4 set shiftwidth=4 */
