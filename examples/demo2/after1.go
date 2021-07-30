package main

import "time"

// 62784  after1       213.1     00:15.72 10/2   0    19    6724K+ 0B
// gc回收

func main() {
	for {
		<-time.After(10 * time.Nanosecond)
	}
}
