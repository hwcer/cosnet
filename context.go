package cosnet

import (
	"github.com/hwcer/cosnet/message"
	"github.com/hwcer/logger"
)

type Context struct {
	*Socket
	Message message.Message
}

func (this *Context) UUID() (r string) {
	if p := this.Socket.Player(); p != nil {
		r = p.UUID()
	}
	return
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
	return this.Socket.Send(this.Path(), err)
}

// Errorf 使用当前路径向客户端写入一个特定错误码的信息
func (this *Context) Errorf(format any, args ...any) error {
	txt := logger.Sprintf(format, args)
	return this.Socket.Send(this.Path(), txt)
}
