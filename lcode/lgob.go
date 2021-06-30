package lcode

import (
	"bufio"
	"encoding/gob"
	"io"
	"lrpc/log"
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

func (gc *GobCodec) ReadHeader(h *Header) error {
	return gc.dec.Decode(h)
}

func (gc *GobCodec) ReadBody(body interface{}) error {
	return gc.dec.Decode(body)
}

func (gc *GobCodec) Write(h *Header, body interface{}) (err error) {
	defer func() {
		_ = gc.buf.Flush()
		if err != nil {
			_ = gc.Close()
		}
	}()

	if err = gc.enc.Encode(h); err != nil {
		log.Errorf("rpc codec: gob error encoding header:", err)
		return
	}

	if err = gc.enc.Encode(body); err != nil {
		log.Errorf("rpc codec: gob error encoding body:", err)
		return
	}

	return
}

func (gc *GobCodec) Close() error {
	return gc.conn.Close()
}

/* vim: set tabstop=4 set shiftwidth=4 */
