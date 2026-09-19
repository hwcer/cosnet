package tcp

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/hwcer/cosnet/listener"
	"github.com/hwcer/cosnet/message"
)

func NewConn(c net.Conn) *Conn {
	return &Conn{Conn: c}
}

type Conn struct {
	net.Conn
	head []byte
	buff *bytes.Buffer
	wmu  sync.Mutex //WriteMessage 的 buff 构建与写盘必须串行:框架内单写协程安全,但本方法是导出接口
}

func (this *Conn) ReadMessage(_ listener.Socket, msg message.Message) error {
	if this.head == nil {
		this.head = message.Options.Head()
	}
	var err error
	if _, err = io.ReadFull(this.Conn, this.head); err != nil {
		return err
	}
	err = msg.Parse(this.head)
	if err != nil {
		return fmt.Errorf("READ HEAD ERR,RemoteAddr:%v,HEAD:%v ,ERR:%v", this.RemoteAddr().String(), this.head, err)
	}
	return this.readMsgTrue(msg)

}
func (this *Conn) readMsgTrue(msg message.Message) (err error) {
	if msg.Size() == 0 {
		return nil
	}
	_, err = msg.Write(this.Conn)
	if err != nil {
		return fmt.Errorf("READ BODY ERR:%v", err)
	}
	return nil
}

func (this *Conn) WriteMessage(_ listener.Socket, msg message.Message) error {
	this.wmu.Lock()
	defer this.wmu.Unlock()
	if this.buff == nil {
		this.buff = new(bytes.Buffer)
	}
	defer func() {
		this.buff.Reset()
	}()
	var err error
	if _, err = msg.Bytes(this.buff, true); err != nil {
		return err
	}
	_, err = this.Conn.Write(this.buff.Bytes())
	return err
}
