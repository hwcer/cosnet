package cosnet

import (
	"fmt"
)

type HandlerFunc func(*Socket, *Message) error

func (this *Cosnet) call(sock *Socket, msg *Message) (err error) {
	if msg.Code == 0 {
		return
	}
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()

	if fn, ok := this.handle[msg.Code]; ok {
		err = fn(sock, msg)
	} else if this.Handler != nil {
		err = this.Handler(sock, msg)
	} else {
		err = fmt.Errorf("无法识别的消息:%v", msg.Code)
	}
	return
}

func (this *Cosnet) Register(code uint16, fun HandlerFunc) {
	this.handle[code] = fun
}
