package lcode

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"errors"
	"io"

	"github.com/zulong210220/lrpc/log"
)

type GobCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *gob.Decoder
	enc  *gob.Encoder
}

var (
	_ Codec = (*GobCodec)(nil)
)

func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)

	return &GobCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(buf),
	}
}

// ---

const (
	BUF_SIZE = 1024 * 1024
)

func (gc *GobCodec) ReadHeader(h *Header) error {
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

	b := bytes.NewBuffer(data)
	dec := gob.NewDecoder(b)
	return dec.Decode(&h)
}

func (gc *GobCodec) Read(msg *Message) error {
	return nil
}

func (gc *GobCodec) Decode(data []byte, body interface{}) error {
	b := bytes.NewBuffer(data)
	dec := gob.NewDecoder(b)
	return dec.Decode(body)
}

func (gc *GobCodec) ReadBody(body interface{}) error {
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
	dec := gob.NewDecoder(b)
	return dec.Decode(body)
}

func (gc *GobCodec) Write(h *Header, body interface{}) (err error) {
	defer func() {
		_ = gc.buf.Flush()
		if err != nil {
			_ = gc.Close()
		}
	}()

	var b bytes.Buffer
	enc := gob.NewEncoder(&b)

	if err = enc.Encode(h); err != nil {
		log.Errorf("gob.Encode.Write", "rpc codec: gob error encoding header:%v", err)
		return
	}

	if err = enc.Encode(body); err != nil {
		log.Errorf("gob.Encode.Write", "rpc codec: gob error encoding body:%v", err)
		return
	}
	var n int
	n, err = gc.conn.Write(b.Bytes())
	if err != nil {
		log.Errorf("gob.Write", "rpc codec: gob error write:%d buffer:%v", n, err)
		return
	}

	return
}

func (gc *GobCodec) Close() error {
	return gc.conn.Close()
}

/* vim: set tabstop=4 set shiftwidth=4 */
