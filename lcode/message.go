package lcode

import (
	"bytes"
	"encoding/binary"

	"github.com/zulong210220/lrpc/log"
)

type Message struct {
	H *Header
	B []byte
}

func (m *Message) Pack() ([]byte, error) {

	dataBuf := bytes.NewBuffer([]byte{})
	var err error

	n := uint32(len(m.H.ServiceMethod))

	err = binary.Write(dataBuf, binary.BigEndian, n)
	if err != nil {
		log.Errorf("Message.Pack", " binary.Write len ServiceMethod failed err:%v", err)
		return nil, err
	}

	err = binary.Write(dataBuf, binary.BigEndian, []byte(m.H.ServiceMethod))
	if err != nil {
		log.Errorf("Message.Pack", " binary.Write ServiceMethod failed err:%v", err)
		return nil, err
	}

	err = binary.Write(dataBuf, binary.BigEndian, m.H.Seq)
	if err != nil {
		log.Errorf("Message.Pack", " binary.Write seq failed err:%v", err)
		return nil, err
	}

	n = uint32(len(m.H.Error))
	err = binary.Write(dataBuf, binary.BigEndian, n)
	if err != nil {
		log.Errorf("Message.Pack", " binary.Write len Error failed err:%v", err)
		return nil, err
	}

	if n > 0 {
		err = binary.Write(dataBuf, binary.BigEndian, []byte(m.H.Error))
		if err != nil {
			log.Errorf("Message.Pack", " binary.Write Error failed err:%v", err)
			return nil, err
		}
	}

	n = uint32(len(m.B))
	err = binary.Write(dataBuf, binary.BigEndian, n)
	if err != nil {
		log.Errorf("Message.Pack", " binary.Write len Body failed err:%v", err)
		return nil, err
	}

	err = binary.Write(dataBuf, binary.BigEndian, m.B)
	if err != nil {
		log.Errorf("Message.Pack", " binary.Write Body failed err:%v", err)
		return nil, err
	}

	data := dataBuf.Bytes()
	return data, err
}

func (m *Message) Unpack(data []byte) error {
	dataBuf := bytes.NewReader(data)
	var (
		n   uint32
		err error
	)

	err = binary.Read(dataBuf, binary.BigEndian, &n)
	if err != nil {
		log.Errorf("Message.Unpack", " binary.Read len ServiceMethod failed err:%v", err)
		return err
	}

	buf := make([]byte, n)
	err = binary.Read(dataBuf, binary.BigEndian, &buf)
	if err != nil {
		log.Errorf("Message.Unpack", " binary.Read ServiceMethod failed err:%v", err)
		return err
	}

	m.H.ServiceMethod = string(buf)

	err = binary.Read(dataBuf, binary.BigEndian, &m.H.Seq)
	if err != nil {
		log.Errorf("Message.Unpack", " binary.Read seq failed err:%v", err)
		return err
	}

	err = binary.Read(dataBuf, binary.BigEndian, &n)
	if err != nil {
		log.Errorf("Message.Unpack", " binary.Read len Error failed err:%v", err)
		return err
	}

	if n > 0 {
		buf = make([]byte, n)
		err = binary.Read(dataBuf, binary.BigEndian, &buf)
		if err != nil {
			log.Errorf("Message.Unpack", " binary.Read Error failed err:%v", err)
			return err
		}
		m.H.Error = string(buf)
	}

	err = binary.Read(dataBuf, binary.BigEndian, &n)
	if err != nil {
		log.Errorf("Message.Unpack", " binary.Read len Body failed err:%v", err)
		return err
	}

	buf = make([]byte, n)
	err = binary.Read(dataBuf, binary.BigEndian, &buf)
	if err != nil {
		log.Errorf("Message.Unpack", " binary.Read Body failed err:%v", err)
		return err
	}
	m.B = buf

	return err
}
