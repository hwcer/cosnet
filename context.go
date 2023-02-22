package cosnet

import (
	"github.com/hwcer/cosgo/utils"
	"github.com/hwcer/cosgo/values"
)

type Context struct {
	*Message
	Socket *Socket
	//Binder binder.Interface //编码解码方式
}

func (this *Context) Bind(i interface{}) error {
	return this.Message.Unmarshal(i, Binder)
}

// Send 发送消息,path默认为当前路径
func (this *Context) Send(data interface{}, path ...string) (err error) {
	var p string
	if len(path) > 0 {
		p = path[0]
	} else {
		p = this.Message.Path()
	}
	return this.Socket.Send(0, p, data)
}

// Error 使用当前路径向客户端写入一个默认错误码的信息
func (this *Context) Error(err interface{}) error {
	return this.Socket.Send(int16(values.MessageErrorCodeDefault), this.Message.Path(), err)
}

// Errorf 使用当前路径向客户端写入一个特定错误码的信息
func (this *Context) Errorf(code int16, format interface{}, args ...interface{}) error {
	txt := utils.Sprintf(format, args)
	if code == 0 {
		code = int16(values.MessageErrorCodeDefault)
	}
	return this.Socket.Send(code, this.Message.Path(), txt)
}

// Acquire 获取一个消息体,如果中途 放弃使用（没有使用 socket.write(msg)）记得归还
//func (this *Context) Acquire(code int16, path string, data interface{}) (msg *Message, err error) {
//	msg = this.Socket.Sockets.Acquire()
//	defer func() {
//		if err != nil {
//			this.Socket.Sockets.Release(msg)
//		}
//	}()
//	err = msg.Marshal(code, path, data, this.Binder)
//	return
//}
