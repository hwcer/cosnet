package cosnet

import (
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

func (this *Context) Send(path string, data any) error {
	i := this.Message.Index()
	return this.Socket.Send(i, path, data)
}
func (this *Context) Write(m message.Message) error {
	return this.Socket.Write(m)
}

// Reply 使用当前路径回复
func (this *Context) Reply(v any) (err error) {
	p := Options.S2CConfirm
	if p == "" {
		p, _, err = this.Message.Path()
	}
	if err != nil {
		return err
	}
	return this.Send(p, v)
}

// Error 使用当前路径向客户端写入一个默认错误码的信息
func (this *Context) Error(err any) {
	Errorf(this.Socket, err)
}

// Errorf 使用当前路径向客户端写入一个特定错误码的信息
func (this *Context) Errorf(format any, args ...any) {
	Errorf(this.Socket, format, args...)
}
