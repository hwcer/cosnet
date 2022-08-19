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

//type Agents interface {
//	CGO(func(ctx context.Context))
//	Emit(e EventType, s *Socket, attach ...interface{}) (r bool)
//	Errorf(s *Socket, format interface{}, args ...interface{}) (r bool)
//	Handler() Handler
//}

type Handler interface {
	Head() int                    //Head size
	Call(*Socket, Message) bool   //执行消息,返回false会踢下线
	Acquire() Message             //获取message
	Register(i interface{}) error //注册协议
}

type Message interface {
	Size() int                            //包体总长
	Code() interface{}                    //协议号，或者说路径
	Data() []byte                         //包体原始二进制
	Parse(head []byte) error              //使用二进制head填充包头
	Bytes() (b []byte, err error)         //消息转换成二进制
	Write(r io.Reader) (n int, err error) //从conn中读取数据写入到data
	Release()                             //消息不再使用释放消息,主要用于poll消息缓存池
	Marshal(code, body interface{}) error //使用code, body 填充message
	Unmarshal(i interface{}) error        //解析Message body
}
