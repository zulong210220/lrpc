package models

import (
	"fmt"
	"time"
)

type Foo int

type Args struct {
	Num1, Num2 int
}

func (a *Args) Reset() {

}

func (a *Args) String() string {
	return ""
}

func (a *Args) ProtoMessage() {

}

type Reply struct {
	Num int
}

func (a *Reply) Reset() {

}

func (a *Reply) String() string {
	return fmt.Sprintf("[%d]", a.Num)
}

func (a *Reply) ProtoMessage() {

}

func (f Foo) Sum(args Args, reply *Reply) error {
	(*reply).Num = args.Num1 + args.Num2
	return nil
}

func (f Foo) sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func (f Foo) Timeout(argv int, reply *int) error {
	time.Sleep(2 * time.Second)
	*reply = 999
	return nil
}

func (f *Foo) Reset() {

}
func (f *Foo) String() string {
	return ""
}

func (f *Foo) ProtoMessage() {

}
