package cosnet

type Context struct {
	*Message
	Socket *Socket
}

func (this *Context) Write(code int32, path string, data interface{}) error {
	msg := this.Acquire()
	if err := msg.Marshal(code, path, data); err != nil {
		return err
	}
	_ = this.Socket.Write(msg)
	return nil
}

// Reply 使用当前消息路径回复信息
func (this *Context) Reply(code int32, data interface{}) error {
	path := this.Path()
	return this.Write(code, path, data)
}

// Acquire 获取一个空消息体
func (this *Context) Acquire() *Message {
	return this.Socket.Agents.Acquire()
}
