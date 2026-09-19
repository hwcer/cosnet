package wss

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hwcer/cosnet/listener"
	"github.com/hwcer/cosnet/message"
	"github.com/hwcer/logger"
)

func NewConn(c *websocket.Conn) *Conn {
	//读帧上限:gorilla 会把整帧读进内存才交给 Transform 校验,不设限的话
	//单个恶意客户端用超大帧声明即可打爆内存(游戏服公网直达)
	c.SetReadLimit(int64(message.Options.MaxDataSize) + 4096)
	return &Conn{Conn: c}
}

// Conn net.Conn
type Conn struct {
	*websocket.Conn
	buff *bytes.Buffer
	wmu  sync.Mutex // gorilla/websocket要求同一时刻仅允许一个写者,此锁序列化所有写入路径
}

// WriteRaw 直写原始websocket帧,绕过transform序列化
// 供SocketIO等底层协议握手/心跳回包使用,与WriteMessage共用写锁
func (c *Conn) WriteRaw(messageType int, data []byte) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	return c.Conn.WriteMessage(messageType, data)
}

// Read 实现 net.Conn 接口,不推荐使用
func (c *Conn) Read(b []byte) (int, error) {
	return 0, errors.New("wss conn Read not support")
}

// Write 实现 net.Conn  接口, 不推荐使用
func (c *Conn) Write(b []byte) (n int, err error) {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	err = c.Conn.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (c *Conn) SetDeadline(t time.Time) error {
	if err := c.Conn.SetReadDeadline(t); err != nil {
		return err
	}
	return c.Conn.SetWriteDeadline(t)
}

// ReadMessage 实现cosnet新版本的接口
// 返回错误时会关闭连接
func (c *Conn) ReadMessage(socket listener.Socket, msg message.Message) error {
	t, b, err := c.Conn.ReadMessage()
	if err != nil {
		if websocket.IsUnexpectedCloseError(err) {
			log.Printf("error: %v", err)
		}
		return err
	}
	if t == websocket.CloseMessage {
		return net.ErrClosed
	}
	if t != websocket.BinaryMessage && t != websocket.TextMessage {
		return errors.New("wss conn ReadMessage not support")
	}
	if len(b) == 0 {
		return io.EOF
	}
	if err = Options.Transform.ReadMessage(socket, msg, b); err != nil {
		return err
	}
	return nil
}

func (c *Conn) WriteMessage(socket listener.Socket, msg message.Message) error {
	//buff 构建与 Reset 必须全程在写锁内:此前懒初始化/Reset 在锁外,一旦业务绕过
	//单写协程并发调用本方法,两个协程会并发写同一个 bytes.Buffer(帧交错损坏),
	//一方 Reset 还会截断另一方已取出的切片
	c.wmu.Lock()
	defer c.wmu.Unlock()

	if c.buff == nil {
		c.buff = new(bytes.Buffer)
	}
	defer func() {
		c.buff.Reset()
	}()

	var err error
	if _, err = Options.Transform.WriteMessage(socket, msg, c.buff); err != nil {
		logger.Error(err)
		return err
	}

	b := c.buff.Bytes()
	if len(b) == 0 {
		return nil
	}
	//logger.Trace("Socket response,PATH:%v   BODY:%v", msg.Path(), string(b))
	err = c.Conn.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return err
	}
	return nil
}
