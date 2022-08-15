package cosnet

// EventType 事件类型
type EventType uint8

const (
	EventTypeHeartbeat      EventType = iota + 1 //心跳事件
	EventTypeConnected                           //连接成功
	EventTypeDisconnect                          //断开连接
	EventTypeAuthentication                      //身份认证
	EventTypeReplaced                            //被顶号
	EventTypeReconnected                         //重登录
	EventTypeDestroy                             //销毁所有信息
)

type EventsFunc func(*Socket)

func (this *Cosnet) On(e EventType, f EventsFunc) {
	this.listener[e] = append(this.listener[e], f)
}

func (this *Cosnet) Emit(e EventType, s *Socket) {
	for _, f := range this.listener[e] {
		f(s)
	}
}
