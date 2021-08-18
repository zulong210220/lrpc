package lcode

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"

	"github.com/zulong210220/lrpc/log"
)

type JsonCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *json.Decoder
	enc  *json.Encoder
}

var (
	_ Codec = (*JsonCodec)(nil)
)

func NewJsonCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)

	return &JsonCodec{
		conn: conn,
		buf:  buf,
		dec:  json.NewDecoder(conn),
		enc:  json.NewEncoder(buf),
	}
}

// ---

func (gc *JsonCodec) ReadHeader(h *Header) error {
	err := gc.dec.Decode(h)
	return err
}

func (gc *JsonCodec) ReadHeader1(h *Header) error {
	data := make([]byte, 256)
	n, err := gc.conn.Read(data)
	if err != nil {
		log.Error("", "GobCodec.ReadBody Read connection data failed:", err)
		if err == io.EOF {
		}
		return err
	}
	if n == 0 {
		return errors.New("gob read header zero")
	}

	/*
		全部读取了数据
		' {"ServiceMethod":"Foo.Sum","Seq":1,"Error":""}
		{"Num1":1,"Num2":1}
		 '
	*/
	b := bytes.NewBuffer(data)
	dec := json.NewDecoder(b)
	return dec.Decode(&h)
}

func (jc *JsonCodec) Read(msg *Message) error {
	//data := make([]byte, BUF_SIZE)
	// TODO fix
	//var data = make([]byte, 65536)
	var data = make([]byte, 4)
	n, err := jc.conn.Read(data)
	//data, err := ioutil.ReadAll(jc.conn)
	if err != nil {
		log.Errorf("", "JsonCodec.Read connection data n:%d failed err:%v", n, err)
		if err == io.EOF {
			// TODO
			return err
		}
		return err
	}

	total := binary.BigEndian.Uint32(data)

	data = make([]byte, total)
	n, err = jc.conn.Read(data)
	//data, err := ioutil.ReadAll(jc.conn)
	if err != nil {
		log.Errorf("", "JsonCodec.Read connection data n:%d failed:%v", n, err)
		if err == io.EOF {
			// TODO
			return err
		}
	}

	msg.Unpack(data)

	return err
}

func (gc *JsonCodec) ReadBody(body interface{}) error {
	err := gc.dec.Decode(body)
	return err
}

func (gc *JsonCodec) ReadBody1(body interface{}) error {
	data := make([]byte, BUF_SIZE)
	n, err := gc.conn.Read(data)
	if err != nil {
		log.Error("", "GobCodec.ReadBody Read connection data failed:", err)
		if err == io.EOF {
			return nil
		}
		return err
	}
	if n == 0 {
		return errors.New("gob read header zero")
	}

	b := bytes.NewBuffer(data)
	dec := json.NewDecoder(b)
	return dec.Decode(body)
}

func (jc *JsonCodec) Decode(data []byte, body interface{}) error {
	return json.Unmarshal(data, body)
}

func (gc *JsonCodec) Write(h *Header, body interface{}) (err error) {
	defer func() {
		_ = gc.buf.Flush()
		if err != nil {
			_ = gc.Close()
		}
	}()

	var bs []byte
	bs, err = json.Marshal(body)
	if err != nil {
		log.Errorf("JsonCodec.Encode.Marshal", "rpc codec: gob error encoding body:%v", err)
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
	//err = binary.Write(gc.conn, binary.BigEndian, uint32(len(bs)))
	if err != nil {
		log.Errorf("gob.Write", "binary Write len buffer:%v", err)
		return
	}
	tbs := dataBuf.Bytes()
	n, err = gc.conn.Write(tbs)

	n, err = gc.conn.Write(bs)
	if err != nil {
		log.Errorf("gob.Write", "rpc codec: gob error write : %d buffer:%v", n, err)
		return
	}

	return
}

func (gc *JsonCodec) Write1(h *Header, body interface{}) (err error) {
	defer func() {
		_ = gc.buf.Flush()
		if err != nil {
			_ = gc.Close()
		}
	}()

	var n int
	var b bytes.Buffer
	enc := json.NewEncoder(&b)

	if err = enc.Encode(h); err != nil {
		log.Errorf("gob.Encode.Write", "rpc codec: gob error encoding header:%v", err)
		return
	}

	if err = enc.Encode(body); err != nil {
		log.Errorf("gob.Encode.Write", "rpc codec: gob error encoding body:%v", err)
		return
	}
	n, err = gc.conn.Write(b.Bytes())
	if err != nil {
		log.Errorf("gob.Write", "rpc codec: gob error write :%d buffer:%v", n, err)
		return
	}

	return
}

func (gc *JsonCodec) Close() error {
	return gc.conn.Close()
}

/* vim: set tabstop=4 set shiftwidth=4 */
