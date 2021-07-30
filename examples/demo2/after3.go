package main

import "time"

// 63715  after3       138.1     01:37.45 10     0    19    5047M- 0B

func main() {
	for {
		select {
		case <-time.After(3 * time.Second):
		default:
		}
	}
}
