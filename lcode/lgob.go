package lcode

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"io"

	"github.com/zulong210220/lrpc/log"
)

type GobCodec struct {
	conn io.ReadWriteCloser
}

var (
	_ Codec = (*GobCodec)(nil)
)

func NewGobCodec(conn io.ReadWriteCloser) Codec {

	return &GobCodec{
		conn: conn,
	}
}

// ---

func (gc *GobCodec) Read(msg *Message) error {
	fun := "GobCodec.Read"
	// TODO fix
	var data = make([]byte, 4)
	n, err := gc.conn.Read(data)
	if err != nil {
		log.Errorf("GCR", "%s connection total n:%d failed err:%v", fun, n, err)
		if err == io.EOF {
			// TODO
			return err
		}
		return err
	}

	total := binary.BigEndian.Uint32(data)

	data = make([]byte, total)
	n, err = gc.conn.Read(data)
	if err != nil {
		log.Errorf("GCR", "%s connection data n:%d failed err:%v", fun, n, err)
		if err == io.EOF {
			// TODO
			return err
		}
	}

	err = msg.Unpack(data)

	return err
}

func (gc *GobCodec) Decode(data []byte, body IMessage) error {
	b := bytes.NewBuffer(data)
	dec := gob.NewDecoder(b)
	return dec.Decode(body)
}

func (gc *GobCodec) Write(h *Header, body IMessage) (err error) {
	fun := "GobCodec.Write"
	defer func() {
		if err != nil {
			_ = gc.Close()
		}
	}()

	buf := bytes.NewBuffer([]byte{})
	enc := gob.NewEncoder(buf)
	err = enc.Encode(body)
	if err != nil {
		log.Errorf("GEM", "%s rpc codec: gob Encode failed error :%v", fun, err)
		return
	}

	var n int
	bs := buf.Bytes()
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
		log.Errorf("GC", "%s binary Write len buffer:%v", fun, err)
		return
	}

	tbs := dataBuf.Bytes()
	n, err = gc.conn.Write(tbs)
	if err != nil {
		log.Errorf("GC", "%s rpc codec: json error write : %d total :%v", fun, n, err)
		return
	}

	n, err = gc.conn.Write(bs)
	if err != nil {
		log.Errorf("GC", "%s rpc codec: json error write : %d buffer :%v", fun, n, err)
	}

	return
}

func (gc *GobCodec) Close() error {
	return gc.conn.Close()
}

/* vim: set tabstop=4 set shiftwidth=4 */
