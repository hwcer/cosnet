package cosnet

import (
	"fmt"
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosgo/utils"
)

type Context struct {
	*Message
	binder binder.EncodingType //编码解码方式
	Socket *Socket
}

func (this *Context) Bind(i interface{}) error {
	b, err := this.GetBinder()
	if err != nil {
		return err
	}
	return this.Message.Unmarshal(i, b)
}

func (this *Context) Write(code int32, path string, data interface{}) (err error) {
	var msg *Message
	if msg, err = this.Acquire(code, path, data); err == nil {
		_ = this.Socket.Write(msg)
	}
	return
}

// Reply 使用当前消息路径构造新MESSAGE
//func (this *Context) Reply(data interface{}) (re *Message, err error) {
//	re = this.Acquire()
//	err = re.Marshal(0, this.Path(), data)
//	return
//}

// Acquire 获取一个消息体,如果中途 放弃使用（没有使用 socket.write(msg)）记得归还
func (this *Context) Acquire(code int32, path string, data interface{}) (msg *Message, err error) {
	bind, err := this.GetBinder()
	if err != nil {
		return nil, err
	}
	msg = this.Socket.Agents.Acquire()
	defer func() {
		if err != nil {
			this.Socket.Agents.Release(msg)
		}
	}()
	err = msg.Marshal(code, path, data, bind)
	return
}

func (this *Context) Error(err interface{}) *Message {
	return this.Errorf(0, err)
}

// Errorf 构造一个错误消息,code默认-9999
func (this *Context) Errorf(code int32, format interface{}, args ...interface{}) *Message {
	txt := utils.Sprintf(format, args)
	if code == 0 {
		code = -9999
	}
	re, _ := this.Acquire(code, this.Path(), txt)
	return re
}

// SetBinder 设置解码方式
func (this *Context) SetBinder(t binder.EncodingType) {
	this.binder = t
}

// GetBinder 获取编解码方式
func (this *Context) GetBinder() (r binder.Interface, err error) {
	if this.binder == 0 {
		r = this.Socket.Agents.Binder
	} else {
		r = binder.Handle(this.binder)
	}
	if r == nil {
		err = fmt.Errorf("binder not exist")
	}
	return
}
