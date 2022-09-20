package cosnet

type Context struct {
	*Message
	Socket *Socket
}

func (this *Context) Send(path string, data interface{}) error {
	msg := this.Acquire()
	if err := msg.Marshal(path, data); err != nil {
		return err
	}
	_ = this.Socket.Write(msg)
	return nil
}

// Reply 使用当前消息路径回复信息
func (this *Context) Reply(data interface{}) error {
	path := this.data[0:this.code]
	return this.Send(string(path), data)
}

func (this *Context) Acquire() *Message {
	return this.Socket.Agents.Acquire()
}
