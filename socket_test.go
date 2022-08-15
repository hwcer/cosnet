package cosnet

import (
	"fmt"
	"testing"
)

var ch = make(chan int, 1024)

func TestCosnet_Socket(t *testing.T) {

	ch <- 1
	ch <- 2
	ch <- 3
	close(ch)

	fmt.Printf("通道数据：%v\n", len(ch))
	read()
	write()
}

func write() {
	defer func() {
		_ = recover()
	}()
	select {
	case ch <- 10:
		fmt.Printf(" 成功写入通道\n")
	default:
		fmt.Printf(" 通道已满无法写消息\n")
	}
}

func read() {
	for {
		select {
		case x := <-ch:
			if x == 0 {
				fmt.Printf("disconnect\n")
				return
			} else {
				fmt.Printf("REV:%v\n", x)
			}
		}
	}
}
