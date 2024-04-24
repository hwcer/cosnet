package tcp

import (
	"bytes"
	"fmt"
	"github.com/hwcer/cosnet/message"
	"io"
	"net"
)

func NewConn(c net.Conn) *Conn {
	return &Conn{Conn: c}
}

type Conn struct {
	net.Conn
	head []byte
	buff *bytes.Buffer
}

func (this *Conn) ReadMessage() (message.Message, error) {
	var err error
	if this.head == nil {
		this.head = message.Options.Head()
	}
	if _, err = io.ReadFull(this.Conn, this.head); err != nil {
		return nil, err
	} else {
		return this.readMsgTrue(this.head)
	}
}
func (this *Conn) readMsgTrue(head []byte) (message.Message, error) {
	//logger.Debug("READ HEAD:%v", head)
	msg := message.Require()
	defer func() {
		message.Release(msg)
	}()
	err := msg.Parse(head)
	if err != nil {
		return nil, fmt.Errorf("READ HEAD ERR,RemoteAddr:%v,HEAD:%v", err, this.RemoteAddr().String(), head)
	}
	//logger.Debug("READ HEAD:%+v BYTE:%v", *msg.Header, head)
	_, err = msg.Write(this.Conn)
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
	if _, err = this.buff.WriteTo(this.Conn); err != nil {
		return
	}
	return
}
