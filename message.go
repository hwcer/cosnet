package cosnet

import (
	"bytes"
	"encoding/binary"
	"github.com/hwcer/logger"
	"io"
)

// Message 默认使用路径模式集中注册协议
// code : 错误码，一般服务器发给客户端，
// path : path路径字节数
// body : body数据字节数
// data : path + body
// path : /ping?t=1  URL路径模式，没有实际属性名，和body共同组成data

const MessageHead = 10

type Message struct {
	code int32  //4    错误码
	path uint16 //2    路径PATH长度
	body uint32 // 4   数据BODY长度
	data []byte //数据   [path][body]
}

// Len 包体总长
func (this *Message) Len() int {
	return int(this.path) + int(this.body)
}
func (this *Message) Code() int32 {
	return this.code
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
	this.code = int32(binary.BigEndian.Uint32(head[0:4]))
	this.path = binary.BigEndian.Uint16(head[4:6])
	this.body = binary.BigEndian.Uint32(head[6:10])
	if this.body > Options.MaxDataSize {
		logger.Debug("包体太长，可能是包头错误,path:%v,body:%v", this.path, this.body)
		return ErrMsgDataSizeTooLong
	}
	return nil
}

// Bytes 生成二进制文件
func (this *Message) Bytes() (b []byte, err error) {
	size := this.Len()
	b = make([]byte, size+MessageHead)
	binary.BigEndian.PutUint32(b[0:4], uint32(this.code))
	binary.BigEndian.PutUint16(b[4:6], this.path)
	binary.BigEndian.PutUint32(b[6:10], this.body)
	if size > 0 {
		copy(b[MessageHead:], this.data[0:size])
	}
	return
}

// Write 从conn中读取数据写入到data
func (this *Message) Write(r io.Reader) (n int, err error) {
	size := this.Len()
	if len(this.data) > size {
		this.data = this.data[0:size]
	} else if len(this.data) < size {
		this.data = make([]byte, size)
	}
	return io.ReadFull(r, this.data)
}

// Marshal 将一个对象放入Message.data
func (this *Message) Marshal(code int32, path string, body interface{}) error {
	this.code = code
	buffer := bytes.NewBuffer(this.data[:0])
	if n, err := buffer.WriteString(path); err == nil {
		this.path = uint16(n)
	} else {
		return err
	}
	if err := Options.MessageBinder.Encode(buffer, body); err != nil {
		return err
	}
	this.body = uint32(buffer.Len() - int(this.path))
	this.data = buffer.Bytes()
	return nil
}

// Unmarshal 解析Message body
func (this *Message) Unmarshal(i interface{}) (err error) {
	return Options.MessageBinder.Unmarshal(this.Body(), i)
}
