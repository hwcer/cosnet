package cosnet

import (
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosnet/message"
)

type Context struct {
	*Socket
	Message message.Message
}

func (this *Context) Path() (string, string, error) {
	return this.Message.Path()
}

func (this *Context) Bind(i any) error {
	return this.Message.Unmarshal(i)
}

func (this *Context) Send(path string, data any) {
	i := this.Message.Index()
	this.Socket.Send(i, path, data)
}
func (this *Context) Write(m message.Message) {
	this.Socket.Write(m)
}

// Error 使用当前路径向客户端写入一个默认错误码的信息
func (this *Context) Error(err any) {
	Errorf(this.Socket, err)
}

// Errorf 使用当前路径向客户端写入一个特定错误码的信息
func (this *Context) Errorf(format any, args ...any) {
	Errorf(this.Socket, format, args...)
}
func (this *Context) Accept() binder.Binder {
	return this.Message.Binder()
}
