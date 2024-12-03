package cosnet

import (
	"github.com/hwcer/cosgo/values"
	"github.com/hwcer/cosnet/message"
)

type Context struct {
	*Socket
	Message message.Message
}

func (this *Context) Path() string {
	return this.Message.Path()
}
func (this *Context) Bind(i any) error {
	return this.Message.Unmarshal(i)
}

// Reply 使用当前路径回复
func (this *Context) Reply(v any) error {
	return this.Socket.Send(this.Path(), v)
}

// Error 使用当前路径向客户端写入一个默认错误码的信息
func (this *Context) Error(err any) error {
	msg := values.Error(err)
	return this.Socket.Send(this.Path(), msg)
}

// Errorf 使用当前路径向客户端写入一个特定错误码的信息
func (this *Context) Errorf(code int, format any, args ...any) error {
	msg := values.Errorf(code, format, args...)
	return this.Socket.Send(this.Path(), msg)
}
