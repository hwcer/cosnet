package cosnet

import (
	"net"
	"sync/atomic"
	"testing"
	"time"
)

// acceptAndClose 模拟"连上即断"的对端：端口能连通，但连接建立后立刻被关闭。
// 现网表现为 pubsub 地址配错端口（该端口上跑的是别的服务）。
func acceptAndClose(t *testing.T) (addr string, accepted *int32, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen error:%v", err)
	}
	accepted = new(int32)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			atomic.AddInt32(accepted, 1)
			_ = conn.Close()
		}
	}()
	return ln.Addr().String(), accepted, func() { _ = ln.Close() }
}

// TestSocketsCloseStopsReconnect Close 之后必须彻底停止重连。
// 回归的是进程退不掉的场景：客户端无限重连 + 对端连上即断，
// 断开->重连->断开会一直转下去，scc.Wait 永远等不到协程结束。
func TestSocketsCloseStopsReconnect(t *testing.T) {
	address, accepted, stop := acceptAndClose(t)
	defer stop()

	ss := New()
	ss.Options.ClientReconnectMax = 0 //无限重连
	ss.Options.ClientReconnectTime = 50
	ss.Options.ClientReconnectMaxDelay = 100
	ss.Options.Heartbeat = 0

	if _, err := ss.Connect(address); err != nil {
		t.Fatalf("connect error:%v", err)
	}
	if err := ss.Start(); err != nil {
		t.Fatalf("start error:%v", err)
	}
	//先确认重连确实在转，否则后面的断言没有意义
	time.Sleep(300 * time.Millisecond)
	if n := atomic.LoadInt32(accepted); n < 2 {
		t.Fatalf("expect reconnect running, accepted:%d", n)
	}

	ss.Close()
	base := atomic.LoadInt32(accepted)
	time.Sleep(500 * time.Millisecond)
	//允许 1 次：Close 时可能有一条已经拨号成功的连接在途，它会被丢弃而不是接管
	if n := atomic.LoadInt32(accepted) - base; n > 1 {
		t.Fatalf("still reconnecting after Close, extra connections:%d", n)
	}
	ss.Range(func(sock *Socket) bool {
		if sock.status != SocketStatusReleased {
			t.Fatalf("socket not released after Close, status:%d", sock.status)
		}
		return true
	})
}

// TestSocketReconnectBackoff 重连必须有退避：对端连上即断时不能变成毫秒级空转。
func TestSocketReconnectBackoff(t *testing.T) {
	address, accepted, stop := acceptAndClose(t)
	defer stop()

	ss := New()
	ss.Options.ClientReconnectMax = 0
	ss.Options.ClientReconnectTime = 100
	ss.Options.ClientReconnectMaxDelay = 200
	ss.Options.Heartbeat = 0

	if _, err := ss.Connect(address); err != nil {
		t.Fatalf("connect error:%v", err)
	}
	if err := ss.Start(); err != nil {
		t.Fatalf("start error:%v", err)
	}
	defer ss.Close()

	time.Sleep(500 * time.Millisecond)
	//首拨 + 500ms 内按 100ms 退避最多再拨 5 次，放宽到 10 次留出调度余量；
	//没有退避时这里会是成百上千
	if n := atomic.LoadInt32(accepted); n > 10 {
		t.Fatalf("reconnect too fast, accepted:%d in 500ms", n)
	}
}
