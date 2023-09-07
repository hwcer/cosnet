package cosnet

import (
	"bytes"
	"encoding/binary"
	"github.com/hwcer/cosgo/binder"
	"io"
)

const (
	MessageHead      = 7
	magicNumber byte = 0x78
)

var MessageByteCap int = 1024 //最小分配内存,越大越浪费内存,但会减少内存分配次数

var Binder binder.Interface = binder.New(binder.MIMEJSON)

func MagicNumber() byte {
	return magicNumber
}

func NewMessage() *Message {
	return &Message{}
}
func MessageHeadSize() int {
	return MessageHead
}

// Message 默认使用路径模式集中注册协议
// path : path路径字节数
// body : body数据字节数
// data : path + body
// path : /ping?t=1  URL路径模式，没有实际属性名，和body共同组成data

type Message struct {
	path uint16 //2    路径PATH长度
	body uint32 //4    数据BODY长度
	data []byte //数据   [path][body]
}

// Len 包体总长
func (this *Message) Len() int {
	return int(this.path) + int(this.body)
}

// Data 包体数据，包括路径和消息体
func (this *Message) Data() []byte {
	return this.data
}

// Path 路径
func (this *Message) Path() string {
	return string(this.data[0:this.path])
}

// Body 消息体数据  [0,,,,10,11,12 .....]
func (this *Message) Body() []byte {
	return this.data[this.path:]
}

// Parse 解析二进制头并填充到对应字段
func (this *Message) Parse(head []byte) error {
	this.path = binary.BigEndian.Uint16(head[1:3])
	this.body = binary.BigEndian.Uint32(head[3:7])
	if this.body > Options.MaxDataSize {
		return ErrMsgDataSizeTooLong
	}
	return nil
}

// Bytes 生成二进制文件
func (this *Message) Bytes(w io.Writer) (n int, err error) {
	size := this.Len()
	head := make([]byte, MessageHead)
	head[0] = magicNumber
	binary.BigEndian.PutUint16(head[1:3], this.path)
	binary.BigEndian.PutUint32(head[3:7], this.body)
	var r int
	if r, err = w.Write(head); err == nil {
		n += r
	} else {
		return
	}
	if size > 0 {
		r, err = w.Write(this.data[0:size])
		n += r
	}
	return
}

// Write 从conn中读取数据写入到data
func (this *Message) Write(r io.Reader) (n int, err error) {
	size := this.Len()
	if cap(this.data) >= size {
		this.data = this.data[0:size]
	} else {
		if size > MessageByteCap {
			this.data = make([]byte, size)
		} else {
			this.data = make([]byte, size, MessageByteCap)
		}
	}
	return io.ReadFull(r, this.data)
}

// Marshal 将一个对象放入Message.data
func (this *Message) Marshal(path string, body any, bind binder.Interface) error {
	if bind == nil {
		bind = Binder
	}
	buffer := bytes.NewBuffer(this.data[:0])
	if n, err := buffer.WriteString(path); err == nil {
		this.path = uint16(n)
	} else {
		return err
	}
	var err error
	switch v := body.(type) {
	case []byte:
		_, err = buffer.Write(v)
	default:
		err = bind.Encode(buffer, body)
	}
	if err != nil {
		return err
	}
	this.body = uint32(buffer.Len() - int(this.path))
	this.data = buffer.Bytes()
	return nil
}

// Unmarshal 解析Message body
func (this *Message) Unmarshal(i interface{}, bind binder.Interface) (err error) {
	if bind == nil {
		bind = Binder
	}
	return bind.Unmarshal(this.Body(), i)
}
