package udp

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/hwcer/cosnet/listener"
	"github.com/hwcer/cosnet/message"
)

// Conn 实现listener.Conn接口
type Conn struct {
	conn    *net.UDPConn
	addr    *net.UDPAddr
	msgChan chan []byte // 用于缓存UDP数据包
	head    []byte      // 用于存储消息头
	ln      *Listener   // 引用监听器，用于在关闭时移除自身
	key     string      // 用于在监听器的conns map中标识自身
}

// Read 从连接中读取数据
func (c *Conn) Read(b []byte) (n int, err error) {
	return n, fmt.Errorf("udp conn read not implemented")
}

// Write 向连接中写入数据
func (c *Conn) Write(b []byte) (n int, err error) {
	n, err = c.conn.WriteToUDP(b, c.addr)
	if err != nil {
		c.ln.removeConn(c.key) // 出错时从活跃连接列表中移除
	}
	return n, err
}

// Close 关闭连接
func (c *Conn) Close() error {
	// 从活跃连接列表中移除
	c.ln.removeConn(c.key)
	// 关闭msgChan通道，避免资源泄漏
	close(c.msgChan)
	// UDP是无连接的，所以这里不需要关闭底层连接
	// 底层连接由Listener管理
	return nil
}

// LocalAddr 返回连接的本地地址
func (c *Conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr 返回连接的远程地址
func (c *Conn) RemoteAddr() net.Addr {
	return c.addr
}

// SetDeadline 设置连接的读写截止时间
func (c *Conn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

// SetReadDeadline 设置连接的读取截止时间
func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline 设置连接的写入截止时间
func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

// ReadMessage 实现cosnet的消息读取接口
func (c *Conn) ReadMessage(_ listener.Socket, msg message.Message) error {
	// 参考TCP实现，使用head字段存储消息头
	if c.head == nil {
		c.head = message.Options.Head()
	}

	// 从msgChan中读取数据包
	b, ok := <-c.msgChan
	if !ok {
		return io.EOF
	}

	// 检查数据包长度是否足够
	if len(b) < len(c.head) {
		return io.ErrUnexpectedEOF
	}

	// 解析消息
	if err := msg.Reset(b); err != nil {
		return err
	}

	return nil
}

// WriteMessage 实现cosnet的消息写入接口
func (c *Conn) WriteMessage(_ listener.Socket, msg message.Message) error {
	// 创建缓冲区
	buffer := bytes.NewBuffer(nil)

	// 生成消息二进制数据，包含头部
	_, err := msg.Bytes(buffer, true)
	if err != nil {
		return err
	}

	// 发送消息
	_, err = c.conn.WriteToUDP(buffer.Bytes(), c.addr)
	return err
}
