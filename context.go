package cosnet

import (
	"fmt"
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosgo/values"
	"github.com/hwcer/cosnet/message"
)

type Context struct {
	*Socket
	binder  binder.Binder
	Message message.Message
}

func (this *Context) Path() (string, error) {
	return this.Message.Path()
}

func (this *Context) Bind(i any) error {
	return this.Message.Unmarshal(i, this.Binder())
}

func (this *Context) Binder() binder.Binder {
	if this.binder != nil {
		return this.binder
	}
	q := this.Message.Query()
	if i, ok := q[binder.ContentType]; ok {
		if t := fmt.Sprintf("%v", i); t != "" {
			this.binder = binder.New(t)
		}
	}
	if this.binder == nil {
		this.binder = message.Binder
	}
	return this.binder
}

func (this *Context) Send(path string, query values.Values, data any) error {
	m := message.Require()
	if err := m.Marshal(path, query, data, this.Binder()); err != nil {
		return err
	}
	return this.Socket.Write(m)
}

// Reply 使用当前路径回复
func (this *Context) Reply(v any) error {
	p, err := message.Reply(this.Message)
	if err != nil {
		return err
	}
	q := this.Message.Query()
	return this.Send(p, q, v)
}

// Error 使用当前路径向客户端写入一个默认错误码的信息
func (this *Context) Error(err any) error {
	return this.Errorf(0, err)
}

// Errorf 使用当前路径向客户端写入一个特定错误码的信息
func (this *Context) Errorf(code int, format any, args ...any) error {
	msg := message.Errorf(code, format, args...)
	return this.Reply(msg)
}
