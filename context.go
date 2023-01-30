package cosnet

import (
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosgo/utils"
)

type Context struct {
	*Message
	Socket *Socket
	Binder binder.Interface //编码解码方式
}

func (this *Context) Bind(i interface{}) error {
	return this.Message.Unmarshal(i, this.Binder)
}

func (this *Context) Write(code int16, path string, data interface{}) (err error) {
	var msg *Message
	if msg, err = this.Acquire(code, path, data); err == nil {
		_ = this.Socket.Write(msg)
	}
	return
}

// Error 使用当前路径向客户端写入一个默认错误码的信息
func (this *Context) Error(err interface{}) error {
	return this.Errorf(0, err)
}

// Errorf 使用当前路径向客户端写入一个特定错误码的信息
func (this *Context) Errorf(code int16, format interface{}, args ...interface{}) error {
	txt := utils.Sprintf(format, args)
	if code == 0 {
		code = -9999
	}
	return this.Write(code, this.Path(), txt)
}

// Acquire 获取一个消息体,如果中途 放弃使用（没有使用 socket.write(msg)）记得归还
func (this *Context) Acquire(code int16, path string, data interface{}) (msg *Message, err error) {
	msg = this.Socket.Agents.Acquire()
	defer func() {
		if err != nil {
			this.Socket.Agents.Release(msg)
		}
	}()
	err = msg.Marshal(code, path, data, this.Binder)
	return
}
