package cosnet

import "github.com/hwcer/cosgo/library/logger"

type HandlerFunc func(*Socket, *Message)

func (this *Cosnet) call(sock *Socket, msg *Message) {
	if msg.Code == 0 {
		return
	}
	if fn, ok := this.handle[msg.Code]; ok {
		fn(sock, msg)
	} else if this.Handler != nil {
		this.Handler(sock, msg)
	} else {
		logger.Debug("无法识别的消息:%v", msg.Code)
	}
}

func (this *Cosnet) Register(code uint16, fun HandlerFunc) {
	this.handle[code] = fun
}
