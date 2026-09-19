package udp

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/hwcer/cosnet/listener"
	"github.com/hwcer/cosnet/message"
)

// Conn 实现listener.Conn接口
type Conn struct {
	conn      *net.UDPConn
	addr      *net.UDPAddr
	msgChan   chan []byte // 用于缓存UDP数据包
	head      []byte      // 用于存储消息头
	ln        *Listener   // 引用监听器，用于在关闭时移除自身
	key       string      // 用于在监听器的conns map中标识自身
	closeOnce sync.Once
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

// Close 关闭连接,幂等:重复调用安全
func (c *Conn) Close() error {
	c.closeOnce.Do(func() {
		// 从活跃连接列表中移除
		c.ln.removeConn(c.key)
		// 关闭msgChan通道，避免资源泄漏
		close(c.msgChan)
		// UDP是无连接的，所以这里不需要关闭底层连接
		// 底层连接由Listener管理
	})
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
//
// 🔴 UDP 是逐包语义且源地址可伪造:解析失败/长度不足的单个数据报必须跳过并继续读
// 下一个,而不是返回错误——旧实现任何坏包都会让 readMsg 整体退出触发 disconnect,
// 攻击者可向在线玩家的地址注入垃圾包将其踢下线
func (c *Conn) ReadMessage(_ listener.Socket, msg message.Message) error {
	// 参考TCP实现，使用head字段存储消息头
	//headSize 仅用于长度比较,直接取常量,不再为每个连接分配一次 10 字节头
	headSize := message.HeadSize()
	for {
		// 从msgChan中读取数据包
		b, ok := <-c.msgChan
		if !ok {
			return io.EOF
		}

		// 检查数据包长度是否足够,不足跳过
		if len(b) < headSize {
			continue
		}

		// 解析消息,失败跳过该包继续读
		if err := msg.Reset(b); err != nil {
			continue
		}
		return nil
	}
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
