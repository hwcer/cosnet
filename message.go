package cosnet

import (
	"bytes"
	"github.com/hwcer/cosgo/utils"
	"github.com/hwcer/logger"
	"io"
	"strings"
)

// Message 默认使用路径模式集中注册协议
// code : 路径字节数
// size : code + len(body)
// data : path + body
// path : /ping?t=1  URL路径模式，没有实际属性名，和body共同组成data

const MessageHead = 6

type Message struct {
	size uint32 //数据BODY 4
	code uint16 //service path len
	data []byte //数据
}

// Size 包体总长
func (this *Message) Size() int {
	return int(this.size)
}

func (this *Message) Data() []byte {
	return this.data
}

// Path 路径
func (this *Message) Path() string {
	path := string(this.data[0:this.code])
	if i := strings.Index(path, "?"); i >= 0 {
		path = path[0:i]
	}
	return path
}

// Query 查询字符串
func (this *Message) Query() string {
	path := string(this.data[0:this.code])
	i := strings.Index(path, "?")
	return path[i+1:]
}

// Parse 解析二进制头并填充到对应字段
func (this *Message) Parse(head []byte) error {
	if err := utils.BytesToInt(head[0:4], &this.size); err != nil {
		return err
	}
	if err := utils.BytesToInt(head[4:6], &this.code); err != nil {
		return err
	}
	if this.size > Options.MaxDataSize {
		logger.Debug("包体太长，可能是包头错误,size:%v,code:%v", this.size, this.code)
		return ErrMsgDataSizeTooLong
	}
	return nil
}

// Bytes 生成二进制文件
func (this *Message) Bytes() (b []byte, err error) {
	buffer := &bytes.Buffer{}
	if err = utils.IntToBuffer(buffer, this.size); err != nil {
		return
	}
	if err = utils.IntToBuffer(buffer, this.code); err != nil {
		return
	}
	if this.size > 0 {
		_, err = buffer.Write(this.data[0:this.size])
	}
	b = buffer.Bytes()
	return
}

// Write 从conn中读取数据写入到data
func (this *Message) Write(r io.Reader) (n int, err error) {
	size := int(this.size)
	if len(this.data) > size {
		this.data = this.data[0:this.size]
	} else if len(this.data) < size {
		this.data = make([]byte, this.size)
	}
	return io.ReadFull(r, this.data)
}

func (this *Message) Reader() io.Reader {
	return bytes.NewReader(this.data)
}

// Reply 使用原始路径填充一个回复信息
func (this *Message) Reply(i interface{}) error {
	path := this.data[0:this.code]
	return this.Marshal(string(path), i)
}

// Marshal 将一个对象放入Message.data
func (this *Message) Marshal(path string, body interface{}) error {
	buffer := bytes.NewBuffer(this.data[:0])
	if n, err := buffer.WriteString(path); err == nil {
		this.code = uint16(n)
		this.size = uint32(n)
	} else {
		return err
	}
	data, err := Options.MessageMarshal(body)
	if err != nil {
		return err
	}
	if n, e := buffer.Write(data); e == nil {
		this.size += uint32(n)
	}
	this.data = buffer.Bytes()
	return nil
}

// Unmarshal 解析Message body
func (this *Message) Unmarshal(i interface{}) error {
	return Options.MessageUnmarshal(this.data[this.code:], i)
}
