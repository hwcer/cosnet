package cosnet

import (
	"context"
	"github.com/hwcer/cosgo/logger"
	"github.com/hwcer/cosgo/scc"
	"time"
)

type Status int8

const (
	StatusTypeConnect Status = iota
	StatusTypeOAuth
	StatusTypeDisconnect
	StatusTypeReleased
)

func (s Status) Disabled() bool {
	return s == StatusTypeReleased
}

type statusMsg struct {
	status Status
	socket *Socket
	handle []func(*Socket)
}

var statusChan chan *statusMsg

// 11v9
// heartbeat 启动协程定时清理无效用户
func heartbeat(ctx context.Context) {
	statusChan = make(chan *statusMsg, 100)
	t := time.Millisecond * time.Duration(Options.SocketHeartbeat)
	ticker := time.NewTimer(t)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case m := <-statusChan:
			doStatusHandle(m)
		case <-ticker.C:
			scc.Try(doHeartbeat)
			ticker.Reset(t)
		}
	}
}

func doHeartbeat(ctx context.Context) {
	Range(func(socket *Socket) bool {
		socket.doHeartbeat()
		return true
	})
}

func doStatusHandle(m *statusMsg) {
	defer func() {
		if r := recover(); r != nil {
			logger.Alert("Recovered in doStatusHandle", r)
		}
	}()
	if m.socket.status == StatusTypeReleased {
		return
	}
	if m.status > 0 {
		m.socket.status = m.status
	}
	for _, f := range m.handle {
		f(m.socket)
	}
}

// doHeartbeat 每一次Heartbeat() heartbeat计数加1
func (sock *Socket) doHeartbeat() {
	t := sock.heartbeat
	switch sock.status {
	case StatusTypeConnect:
		if Options.SocketConnectTime > 0 && t > Options.SocketConnectTime {
			sock.status = StatusTypeReleased
			sock.release()
		}
	case StatusTypeOAuth:
		if Options.SocketConnectTime > 0 && t > Options.SocketConnectTime {
			sock.status = StatusTypeDisconnect
			sock.disconnect()
		}
	case StatusTypeDisconnect:
		if t > Options.SocketDisconnectTime {
			sock.status = StatusTypeReleased
		}
	case StatusTypeReleased:
		sock.release()
	default:

	}
}
func (sock *Socket) GetStatus() Status {
	return sock.status
}

// SetStatus 设置状态
//
//	异步模式 ,后续任务需要在handle中执行
func (sock *Socket) SetStatus(status Status, handle ...func(*Socket)) {
	m := &statusMsg{socket: sock, status: status, handle: handle}
	go func() {
		statusChan <- m
	}()
}

func (sock *Socket) Release(handle ...func(*Socket)) {
	arr := make([]func(*Socket), 0, len(handle)+1)
	arr = append(arr, doReleased)
	arr = append(arr, handle...)
	sock.SetStatus(StatusTypeReleased, arr...)
}

// Disconnect 掉线,包含网络超时，网络错误
func (sock *Socket) Disconnect(handle ...func(*Socket)) {
	arr := make([]func(*Socket), 0, len(handle)+1)
	arr = append(arr, doDisconnect)
	arr = append(arr, handle...)
	sock.SetStatus(StatusTypeDisconnect, arr...)
}

func doReleased(sock *Socket) {
	sock.release()
}

func doDisconnect(sock *Socket) {
	sock.disconnect()
}
