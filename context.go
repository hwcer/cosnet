package cosnet

import (
	"github.com/hwcer/cosnet/message"
)

type Context struct {
	*Socket
	Message message.Message
}

func (this *Context) Path() (string, error) {
	return this.Message.Path()
}

func (this *Context) Bind(i any) error {
	return this.Message.Unmarshal(i)
}

func (this *Context) Send(path string, query map[string]string, data any) error {
	m := message.Require()
	if err := m.Marshal(path, query, data); err != nil {
		return err
	}
	//logger.Debug("发送数据包，path:%v,query:%+v", path, query)
	return this.Socket.Write(m)
}
func (this *Context) Write(m message.Message) error {
	return this.Socket.Write(m)
}

// Reply 使用当前路径回复
func (this *Context) Reply(v any) error {
	p, err := message.Reply(this.Message)
	if err != nil {
		return err
	}
	return this.Send(p, this.Message.Query(), v)
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
