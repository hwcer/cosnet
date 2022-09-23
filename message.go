package cosnet

import (
	"bytes"
	"github.com/hwcer/cosgo/utils"
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
	if err := utils.BytesToInt(head[0:4], &this.code); err != nil {
		return err
	}
	if err := utils.BytesToInt(head[4:6], &this.path); err != nil {
		return err
	}
	if err := utils.BytesToInt(head[6:10], &this.body); err != nil {
		return err
	}
	if this.body > Options.MaxDataSize {
		logger.Debug("包体太长，可能是包头错误,path:%v,body:%v", this.path, this.body)
		return ErrMsgDataSizeTooLong
	}
	return nil
}

// Bytes 生成二进制文件
func (this *Message) Bytes() (b []byte, err error) {
	buffer := &bytes.Buffer{}
	if err = utils.IntToBuffer(buffer, this.code); err != nil {
		return
	}
	if err = utils.IntToBuffer(buffer, this.path); err != nil {
		return
	}
	if err = utils.IntToBuffer(buffer, this.body); err != nil {
		return
	}
	size := this.Len()
	if size > 0 {
		_, err = buffer.Write(this.data[0:size])
	}
	b = buffer.Bytes()
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

func (this *Message) Reader() io.Reader {
	return bytes.NewReader(this.data)
}

// Marshal 将一个对象放入Message.data
func (this *Message) Marshal(code int32, path string, body interface{}) (err error) {
	this.code = code
	buffer := bytes.NewBuffer(this.data[:0])
	var n int
	if n, err = buffer.WriteString(path); err == nil {
		this.path = uint16(n)
	} else {
		return err
	}

	var data []byte
	switch v := body.(type) {
	case []byte:
		data = v
	default:
		data, err = Options.MessageMarshal(body)
	}
	if err != nil {
		return err
	}
	if n, err = buffer.Write(data); err == nil {
		this.body = uint32(n)
	}
	this.data = buffer.Bytes()
	return nil
}

// Unmarshal 解析Message body
func (this *Message) Unmarshal(i interface{}) error {
	return Options.MessageUnmarshal(this.data[this.path:], i)
}
