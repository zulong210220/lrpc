package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type A struct {
	B int64
}

type Option struct {
	Code int32
	Type int64
	a    A
	//d    []byte
}

func main() {
	opt := &Option{
		Code: 9,
		Type: 12,
		//d:    []byte("ping"),
	}
	dataBuf := bytes.NewBuffer([]byte{})
	err := binary.Write(dataBuf, binary.BigEndian, opt)
	fmt.Println("Write", err)
	fmt.Println(dataBuf.Bytes())
}

type Stu struct {
	Age int32
	Id  int32
}

func main2() {
	s := &Stu{
		Age: 21,
		Id:  1,
	}

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, s)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%q\n", buf)
}
