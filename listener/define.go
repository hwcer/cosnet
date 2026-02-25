package listener

import (
	"net"

	"github.com/hwcer/cosgo/session"
	"github.com/hwcer/cosnet/message"
)

// SocketType 定义 Socket 的类型。
type SocketType int8

// Socket 类型常量。
const (
	SocketTypeNone SocketType = iota
	SocketTypeClient
	SocketTypeServer
)

// Conn 定义网络连接接口，扩展了标准库的 net.Conn 接口。
type Conn interface {
	net.Conn
	// ReadMessage 从连接读取消息。
	ReadMessage(Socket, message.Message) error
	// WriteMessage 向连接写入消息。
	WriteMessage(Socket, message.Message) error
}

// Listener 定义网络监听器接口，扩展了标准库的 net.Listener 接口。
type Listener interface {
	// Accept 等待并返回下一个连接。
	Accept() (Conn, error)
	// Close 关闭监听器。
	Close() error
	// Addr 返回监听器的网络地址。
	Addr() net.Addr
}

// Socket 定义 Socket 接口，提供 Socket 的基本操作方法。
type Socket interface {
	// Id 返回 Socket 的唯一标识符。
	Id() uint64
	// Type 返回 Socket 的类型。
	Type() SocketType
	// Data 返回 Socket 绑定的会话数据。
	Data() *session.Data
	// Conn 返回底层网络连接。
	Conn() Conn
	// Send 发送消息到对端。
	Send(flag message.Flag, index int32, path string, data any, safe ...bool) error
	// Errorf 记录错误日志。
	Errorf(format any, args ...any)
	// LocalAddr 返回本地网络地址。
	LocalAddr() net.Addr
	// RemoteAddr 返回远程网络地址。
	RemoteAddr() net.Addr
}
