package cosnet

import (
	"context"
	"errors"
	"fmt"
	"github.com/hwcer/cosgo/storage"
	"github.com/hwcer/cosnet/listener"
	"github.com/hwcer/cosnet/message"
	"github.com/hwcer/scc"
	"io"
	"net"
)

func NewSocket(srv *Server, conn listener.Conn) *Socket {
	socket := &Socket{conn: conn, server: srv}
	socket.stop = make(chan struct{})
	socket.cwrite = make(chan message.Message, Options.WriteChanSize)
	srv.SCC.SGO(socket.readMsg)
	srv.SCC.SGO(socket.writeMsg)
	return socket
}

// Socket 基础网络连接
type Socket struct {
	storage.Data
	conn   listener.Conn
	stop   chan struct{}
	server *Server
	status Status
	cwrite chan message.Message //写入通道,仅仅强制关闭的会被CLOSE
	//netType NetType       //网络连接类型
}

func (sock *Socket) emit(e EventType) bool {
	return sock.server.Emit(e, sock)
}

func (sock *Socket) close() {
	defer func() { _ = recover() }()
	if sock.stop != nil {
		close(sock.stop)
	}
	_ = sock.conn.Close()
}

// disconnect 掉线,包含网络超时，网络错误
func (sock *Socket) disconnect() {
	if sock.status.Disconnect() {
		sock.close()
		sock.KeepAlive()
		sock.emit(EventTypeDisconnect)
	}
}

// Close 强制关闭,无法重连
func (sock *Socket) Close(msg ...message.Message) {
	if !sock.status.Close() || len(msg) == 0 {
		return
	}
	defer func() {
		_ = recover()
	}()
	for _, m := range msg {
		_ = sock.write(m)
	}
}

func (sock *Socket) Player() (r *Player) {
	v := sock.Data.Get()
	if v != nil {
		r, _ = v.(*Player)
	}
	return
}

func (sock *Socket) Errorf(format any, args ...any) {
	sock.server.Errorf(sock, format, args...)
}

// Verified 是否已经登录
func (sock *Socket) Verified() bool {
	return sock.Get() != nil
}

// Heartbeat 每一次Heartbeat() heartbeat计数加1
func (sock *Socket) Heartbeat() {
	heartbeat := sock.status.Heartbeat()
	switch sock.status.status {
	case StatusTypeDestroyed:
		sock.server.Remove(sock)
	case StatusTypeDisconnect:
		if !sock.Verified() || Options.SocketReconnectTime == 0 || heartbeat > Options.SocketReconnectTime {
			sock.server.Remove(sock)
		}
	default:
		if sock.status.closing && (len(sock.cwrite) == 0 || heartbeat >= Options.SocketDestroyingTime) {
			sock.close()
		}
		if heartbeat >= Options.SocketConnectTime {
			sock.disconnect()
		}
	}
}

// KeepAlive 任何行为都清空heartbeat
func (sock *Socket) KeepAlive() {
	sock.status.KeepAlive()
}

func (sock *Socket) LocalAddr() net.Addr {
	if sock.conn != nil {
		return sock.conn.LocalAddr()
	}
	return nil
}
func (sock *Socket) RemoteAddr() net.Addr {
	if sock.conn != nil {
		return sock.conn.RemoteAddr()
	}
	return nil
}

func (sock *Socket) Send(path string, data any) (err error) {
	m := message.Require()
	if err = m.Marshal(path, data); err != nil {
		return
	}
	if err = sock.Write(m); err != nil {
		message.Release(m)
	}
	return
}

// Write 外部写入消息,慎用,注意发送失败时消息回收,参考Send
func (sock *Socket) Write(m message.Message) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()
	if sock.status.Disabled() {
		return ErrSocketClosed
	}
	if !sock.write(m) {
		err = ErrSocketChannelFull
	}
	return
}

func (sock *Socket) write(m message.Message) bool {
	if m == nil || sock.cwrite == nil {
		return true
	}
	select {
	case sock.cwrite <- m:
		return true
	default:
		return false
	}
}

func (sock *Socket) readMsg(_ context.Context) {
	defer sock.disconnect()
	for {
		if msg, err := sock.conn.ReadMessage(); err != nil {
			if err != io.EOF && !errors.Is(err, net.ErrClosed) && !scc.Stopped() {
				sock.Errorf(err)
			}
			return
		} else if msg != nil && !sock.readMsgTrue(msg) {
			return
		}
	}
}
func (sock *Socket) readMsgTrue(msg message.Message) bool {
	defer message.Release(msg)
	if err := msg.Verify(); err != nil {
		sock.Errorf(err)
		return false
	}
	sock.KeepAlive()
	sock.server.handle(sock, msg)
	return true
}

func (sock *Socket) writeMsg(ctx context.Context) {
	defer sock.disconnect()
	for {
		select {
		case <-ctx.Done():
			return
		case <-sock.stop:
			return
		case msg := <-sock.cwrite:
			sock.writeMsgTrue(msg)
		}
	}
}

func (sock *Socket) writeMsgTrue(msg message.Message) {
	var err error
	defer func() {
		message.Release(msg)
		if err != nil {
			sock.Errorf(err)
		} else {
			sock.KeepAlive()
		}
	}()
	err = sock.conn.WriteMessage(msg)
}
