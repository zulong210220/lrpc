package main

import "time"

// 63011  after2       548.6     00:27.69 11/8   0    20    8804K+ 0B

// time.After(d)在d时间之后就会fire，然后被GC回收，不会造成资源泄漏的。

func main() {
	for i := 0; i < 100; i++ {
		go func() {
			for {
				<-time.After(10 * time.Nanosecond)
			}
		}()
	}
	time.Sleep(1 * time.Hour)
}
