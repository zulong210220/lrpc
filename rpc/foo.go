package rpc

import "time"

type Foo int

type Args struct {
	Num1, Num2 int
}

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
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
