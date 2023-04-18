package cosnet

import (
	"github.com/hwcer/cosgo/values"
	"github.com/hwcer/logger"
)

type Context struct {
	*Socket
	Message *Message
}

func (this *Context) UUID() (r string) {
	if p := this.Socket.Player(); p != nil {
		r = p.UUID()
	}
	return
}

func (this *Context) Bind(i any) error {
	return this.Message.Unmarshal(i, Binder)
}

// Error 使用当前路径向客户端写入一个默认错误码的信息
func (this *Context) Error(err any) error {
	return this.Socket.Send(int16(values.MessageErrorCodeDefault), this.Message.Path(), err)
}

// Errorf 使用当前路径向客户端写入一个特定错误码的信息
func (this *Context) Errorf(code int16, format any, args ...any) error {
	txt := logger.Sprintf(format, args)
	if code == 0 {
		code = int16(values.MessageErrorCodeDefault)
	}
	return this.Socket.Send(code, this.Message.Path(), txt)
}
