package lcode

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

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
	fmt.Println("JsonCodec.|", h, "|")
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

	fmt.Println("'", string(data), "'")
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
	data, err := ioutil.ReadAll(jc.conn)
	if err != nil {
		log.Error("", "JsonCodec.Read connection data failed:", err)
		if err == io.EOF {
			return nil
		}
		return err
	}

	dataBuf := bytes.NewReader(data)

	err = binary.Read(dataBuf, binary.LittleEndian, msg.h)
	if err != nil {
		log.Error("", "JsonCodec.Read binary connection data failed:", err)
		return err
	}

	err = binary.Read(dataBuf, binary.LittleEndian, msg.b)
	if err != nil {
		log.Error("", "JsonCodec.Read binary connection body data failed:", err)
		return err
	}

	return err
}

func (gc *JsonCodec) ReadBody(body interface{}) error {
	err := gc.dec.Decode(body)
	fmt.Println("JsonCodec.RBody.|", body, "|")
	return err
}

func (gc *JsonCodec) ReadBody1(body interface{}) error {
	data := make([]byte, BUF_SIZE)
	log.Infof("before json.ReadBody", "read body ")
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

	log.Infof("gob.ReadBody", "read body %s", data)
	b := bytes.NewBuffer(data)
	dec := json.NewDecoder(b)
	return dec.Decode(body)
}

func (gc *JsonCodec) Write(h *Header, body interface{}) (err error) {
	defer func() {
		_ = gc.buf.Flush()
		if err != nil {
			_ = gc.Close()
		}
	}()

	dataBuf := bytes.NewBuffer([]byte{})
	var n int

	err = binary.Write(dataBuf, binary.LittleEndian, h)
	if err != nil {
		log.Errorf("JsonCodec.Write", "rpc codec: gob error encoding header:%v", err)
		return
	}

	err = binary.Write(dataBuf, binary.LittleEndian, body)
	if err != nil {
		log.Errorf("JsonCodec.Encode.Write", "rpc codec: gob error encoding body:%v", err)
		return
	}

	n, err = gc.conn.Write(dataBuf.Bytes())
	if err != nil {
		log.Errorf("gob.Write", "rpc codec: gob error write buffer:%v", err)
		return
	}
	log.Infof("gob.Write", "conn write n:%d", n)

	return
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
		log.Errorf("gob.Write", "rpc codec: gob error write buffer:%v", err)
		return
	}
	log.Infof("gob.Write", "conn write n:%d", n)

	return
}

func (gc *JsonCodec) Close() error {
	return gc.conn.Close()
}

/* vim: set tabstop=4 set shiftwidth=4 */
