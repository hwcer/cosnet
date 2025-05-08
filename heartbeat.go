package cosnet

import (
	"context"
	"github.com/hwcer/cosgo/scc"
	"time"
)

func stop() {
	for _, l := range instance {
		_ = l.Close()
	}
}

// heartbeat 启动协程定时清理无效用户
func heartbeat(ctx context.Context) {
	t := time.Millisecond * time.Duration(Options.SocketHeartbeat)
	ticker := time.NewTimer(t)
	defer ticker.Stop()
	defer stop()
	for {
		select {
		case <-ctx.Done():
			return
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

// doHeartbeat 每一次Heartbeat() heartbeat计数加1
func (sock *Socket) doHeartbeat() {
	if Options.SocketConnectTime > 0 && sock.heartbeat > Options.SocketConnectTime {
		sock.disconnect()
	} else {
		sock.heartbeat += Options.SocketHeartbeat
	}
}
