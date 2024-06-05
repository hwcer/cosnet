package tcp

import (
	"bytes"
	"fmt"
	"github.com/hwcer/cosnet/message"
	"github.com/hwcer/logger"
	"net"
)

func NewConn(c net.Conn) *Conn {
	conn := &Conn{}
	conn.TCPConn = c.(*net.TCPConn)
	if err := conn.TCPConn.SetLinger(0); err != nil {
		logger.Alert(err)
	}
	//_ = conn.TCPConn.SetReadBuffer(int(message.Options.MaxDataSize) * 10)
	//_ = conn.TCPConn.SetWriteBuffer(int(message.Options.MaxDataSize) * 10)
	return conn
}

type Conn struct {
	*net.TCPConn
	head []byte
	buff *bytes.Buffer
}

func (this *Conn) ReadMessage() (message.Message, error) {
	var err error
	if this.head == nil {
		this.head = message.Options.Head()
	}
	if _, err = this.TCPConn.Read(this.head); err != nil {
		return nil, err
	} else {
		return this.readMsgTrue(this.head)
	}
}
func (this *Conn) readMsgTrue(head []byte) (message.Message, error) {
	msg := message.Require()
	err := msg.Parse(head)
	if err != nil {
		return nil, fmt.Errorf("READ HEAD ERR,RemoteAddr:%v,HEAD:%v ,ERR:%v", this.RemoteAddr().String(), head, err)
	}
	if msg.Size() == 0 {
		return nil, nil
	}
	_, err = msg.Write(this.TCPConn)
	if err != nil {
		return nil, fmt.Errorf("READ BODY ERR:%v", err)
	}
	return msg, nil
}

func (this *Conn) WriteMessage(msg message.Message) (err error) {
	if this.buff == nil {
		this.buff = new(bytes.Buffer)
	}
	defer func() {
		this.buff.Reset()
	}()
	if _, err = msg.Bytes(this.buff, true); err != nil {
		return
	}
	_, err = this.TCPConn.Write(this.buff.Bytes())
	return
}
