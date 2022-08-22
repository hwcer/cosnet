package sockets

import (
	"io"
)

type NetType uint8

// NetType
const (
	NetTypeClient NetType = 1 //client Request
	NetTypeServer         = 2 //Server Listener
)

type Handler interface {
	Head() int                    //Head size
	Handle(*Socket, Message) bool //执行消息,返回false会踢下线
	Acquire() Message             //获取message
	Release(Message)              //消息不再使用释放消息,主要用于poll消息缓存池
}

type Message interface {
	Size() int                            //包体总长
	Parse(head []byte) error              //使用二进制head填充包头
	Bytes() (b []byte, err error)         //消息转换成二进制
	Write(r io.Reader) (n int, err error) //从conn中读取body数据
}
