package wss

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hwcer/cosnet/listener"
	"github.com/hwcer/cosnet/message"
	"github.com/hwcer/logger"
)

func NewConn(c *websocket.Conn) *Conn {
	return &Conn{Conn: c}
}

// Conn net.Conn
type Conn struct {
	*websocket.Conn
	buff *bytes.Buffer
}

// Read 实现 net.Conn 接口,不推荐使用
func (c *Conn) Read(b []byte) (int, error) {
	return 0, errors.New("wss conn Read not support")
}

// Read 实现 net.Conn  接口, 不推荐使用
func (c *Conn) Write(b []byte) (n int, err error) {
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
	err = c.Conn.WriteMessage(websocket.TextMessage, b)
	if err != nil {
		return err
	}
	return nil
}
