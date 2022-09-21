package cosnet

import (
	"fmt"
)

// EventType 事件类型
type EventType uint8

const (
	EventTypeError       EventType = iota + 1 //用户级别错误
	EventTypeHeartbeat                        //心跳事件
	EventTypeConnected                        //连接成功
	EventTypeVerified                         //身份认证
	EventTypeDisconnect                       //断开连接
	EventTypeReconnected                      //重登录
	EventTypeReplaced                         //被顶号
	EventTypeDestroyed                        //销毁所有信息
)

type EventsFunc func(*Socket, interface{}) bool

func (this *Agents) On(e EventType, f EventsFunc) {
	this.listener[e] = append(this.listener[e], f)
}

func (this *Agents) Emit(e EventType, s *Socket, attach ...interface{}) (r bool) {
	var v interface{}
	if len(attach) > 0 {
		v = attach[0]
	}
	for _, f := range this.listener[e] {
		if r = f(s, v); !r {
			return r
		}
	}
	return true
}

// Errorf 抛出一个异常
func (this *Agents) Errorf(s *Socket, format interface{}, args ...interface{}) (r bool) {
	if len(args) == 0 {
		return this.Emit(EventTypeError, s, format)
	}
	v, ok := format.(string)
	if !ok {
		v = fmt.Sprintf("%v", format)
	}
	err := fmt.Errorf(v, args...)
	return this.Emit(EventTypeError, s, err)
}