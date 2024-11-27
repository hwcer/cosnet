package cosnet

import (
	"context"
	"errors"
	"fmt"
	"github.com/hwcer/cosgo/logger"
	"github.com/hwcer/cosgo/scc"
	"github.com/hwcer/cosgo/storage"
	"github.com/hwcer/cosnet/listener"
	"github.com/hwcer/cosnet/message"
	"io"
	"net"
)

func NewSocket(srv *Server, conn listener.Conn) *Socket {
	socket := &Socket{conn: conn, Server: srv}
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
	cwrite chan message.Message //写入通道,仅仅强制关闭的会被CLOSE
	Server *Server
	Status Status
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
	if sock.Status.Disconnect() {
		sock.close()
		sock.KeepAlive()
		sock.Emit(EventTypeDisconnect)
	}
}

func (sock *Socket) Emit(e EventType) bool {
	return sock.Server.Emit(e, sock)
}

// Close 强制关闭,无法重连
func (sock *Socket) Close(msg ...message.Message) {
	if !sock.Status.Close() || len(msg) == 0 {
		return
	}
	defer func() {
		_ = recover()
	}()
	for _, m := range msg {
		_ = sock.write(m)
	}
}

//func (sock *Socket) Values() (r any) {
//	return sock.Data.Get()
//}

func (sock *Socket) Errorf(format any, args ...any) {
	sock.Server.Errorf(sock, format, args...)
}

// Heartbeat 每一次Heartbeat() heartbeat计数加1
func (sock *Socket) Heartbeat() {
	heartbeat := sock.Status.Heartbeat()
	switch sock.Status.status {
	case StatusTypeDestroyed:
		sock.Server.Remove(sock)
	case StatusTypeDisconnect:
		if sock.Get() == nil || Options.SocketReconnectTime == 0 || heartbeat > Options.SocketReconnectTime {
			sock.Server.Remove(sock)
		}
	default:
		if sock.Status.closing && (len(sock.cwrite) == 0 || heartbeat >= Options.SocketDestroyingTime) {
			sock.close()
		}
		if heartbeat >= Options.SocketConnectTime {
			sock.disconnect()
		}
	}
}

// KeepAlive 任何行为都清空heartbeat
func (sock *Socket) KeepAlive() {
	sock.Status.KeepAlive()
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
		logger.Alert("socket Send Error:%v", err)
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
	if sock.Status.Disabled() {
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
		} else if !sock.readMsgTrue(msg) {
			return
		}
	}
}
func (sock *Socket) readMsgTrue(msg message.Message) bool {
	if msg == nil {
		return true
	}
	defer message.Release(msg)
	if err := msg.Verify(); err != nil {
		sock.Errorf(err)
		return false
	}
	sock.KeepAlive()
	sock.Server.handle(sock, msg)
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
