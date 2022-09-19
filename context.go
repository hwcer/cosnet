package cosnet

import "net/url"

type Context struct {
	query   url.Values
	Socket  *Socket
	Message *Message
}

func (this *Context) Path() string {
	return this.Message.Path()
}

func (this *Context) Query(key string) string {
	if this.query == nil {
		s := this.Message.Query()
		if v, err := url.ParseQuery(s); err == nil {
			this.query = v
		} else {
			this.query = url.Values{}
		}
	}
	return this.query.Get(key)
}

func (this *Context) Write(path string, data interface{}) error {
	msg := this.Socket.Agents.Acquire()
	if err := msg.Marshal(path, data); err != nil {
		return err
	}
	_ = this.Socket.Write(msg)
	return nil
}

func (this *Context) Unmarshal(i interface{}) error {
	return this.Message.Unmarshal(i)
}
