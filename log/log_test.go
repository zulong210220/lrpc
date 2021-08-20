package log

import (
	"fmt"
	"testing"
	"time"
)

func initTest() {
	c := &Config{
		Dir:      "./logs",
		FileSize: 256,
		FileNum:  100,
		Env:      "test",
		Level:    "INFO",
		FileName: "testing",
	}

	Init(c)
}

func TestA(t *testing.T) {
	initTest()

	Info("ok")
	Infof("", "%s:%d:%f", "aaa", 123, 3.5)
	Warning("ok")

	ForceFlush()
	time.Sleep(10 * time.Millisecond)
}

func TestFmt(t *testing.T) {
	bb := make([]byte, 16)
	bbb := &bb
	total := 9
	data := (*bbb)[:total]
	fmt.Println(len(data))
}

/* vim: set tabstop=4 set shiftwidth=4 */
