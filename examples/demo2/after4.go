package main

import "time"

// 64091  after4       112.1     00:13.50 10/1   0    19    6396K+ 0B

func main() {
	for {
		t := time.NewTimer(3 * time.Second)
		select {
		case <-t.C:
		default:
			t.Stop()
		}
	}
}
