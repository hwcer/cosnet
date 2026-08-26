package cosnet

import (
	"net"
	"testing"

	"github.com/hwcer/cosnet/tcp"
)

// newTestSocket 造一条真实可用的服务端连接（对端不读不写，只保证 conn 不为 nil）。
func newTestSocket(t *testing.T) (*Socket, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen error:%v", err)
	}
	done := make(chan net.Conn, 1)
	go func() {
		c, e := ln.Accept()
		if e == nil {
			done <- c
		}
	}()
	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial error:%v", err)
	}
	server := <-done

	ss := New()
	ss.Options.Heartbeat = 0 //不启动 daemon，心跳由测试手动推进
	sock, err := ss.Create(tcp.NewConn(server))
	if err != nil {
		t.Fatalf("create error:%v", err)
	}
	return sock, func() {
		_ = client.Close()
		_ = ln.Close()
	}
}

// TestReplacedNegotiation 顶号协商期：只收不发，且倒计时按 SocketReplacedTime 起算。
func TestReplacedNegotiation(t *testing.T) {
	sock, stop := newTestSocket(t)
	defer stop()

	if !sock.CanRead() || !sock.CanWrite() {
		t.Fatalf("fresh socket must be readable and writable")
	}

	if !sock.Replaced("10.0.0.1") {
		t.Fatalf("first Replaced must start the negotiation")
	}
	//协商期的全部意义就在这一行：还能往外发（在途回包/推送），但不再受理新请求
	if sock.CanRead() {
		t.Fatalf("negotiating socket must reject inbound requests")
	}
	if !sock.CanWrite() {
		t.Fatalf("negotiating socket must stay writable, otherwise every in-flight reply is silently dropped")
	}
	if got := sock.Countdown(); got != Options.SocketReplacedTime {
		t.Fatalf("countdown = %d, want %d", got, Options.SocketReplacedTime)
	}
	//重复顶号不得刷新倒计时，否则新端反复重试就能把老连接永远续命、自己永远上不来
	if sock.Replaced("10.0.0.2") {
		t.Fatalf("second Replaced must be a no-op")
	}
	if got := sock.Countdown(); got != Options.SocketReplacedTime {
		t.Fatalf("countdown reset by repeated Replaced: %d", got)
	}
}

// TestCloseNeverExtendsLifetime Close 只缩短剩余存活时间，不给失联连接续命。
// 回归的是：对一条本来还剩 5 秒就会被心跳判死的半开连接发起顶号协商，
// 若倒计时被往回拨，这条死连接反而多活一整个协商期，新端要等更久。
func TestCloseNeverExtendsLifetime(t *testing.T) {
	sock, stop := newTestSocket(t)
	defer stop()

	//模拟已经失联很久：只剩 SocketConnectTime-heartbeat 秒可活
	sock.heartbeat = Options.SocketConnectTime - 5
	if !sock.Replaced("10.0.0.1") {
		t.Fatalf("Replaced failed")
	}
	if got := sock.Countdown(); got != 5 {
		t.Fatalf("countdown = %d, want 5 (Close must not extend lifetime)", got)
	}
}

// TestCloseKeepsWritable 关闭流程中仍可写——这是 Close 注释里"等通道中的消息发送完毕"
// 的实际依据。收紧成 IsReady 的话，Close 一执行后续所有 Write 就全失败。
func TestCloseKeepsWritable(t *testing.T) {
	sock, stop := newTestSocket(t)
	defer stop()

	if !sock.Close(0) {
		t.Fatalf("Close failed")
	}
	if sock.IsReady() {
		t.Fatalf("IsReady must stay strict (connected only)")
	}
	if !sock.CanWrite() {
		t.Fatalf("closing socket must remain writable")
	}
	if sock.CanRead() {
		t.Fatalf("closing socket must not accept new requests")
	}
	if sock.Close(0) {
		t.Fatalf("second Close must be a no-op")
	}
}
