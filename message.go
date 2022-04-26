package cosnet

import (
	"bytes"
	"errors"
	"github.com/hwcer/cosgo/library/logger"
	"github.com/hwcer/cosgo/utils"
)

const HeaderSize = 8

type Header struct {
	size  int32  //数据BODY 4
	Code  uint16 //协议号 2
	Index uint16 //协议编号 2
}

type Message struct {
	*Header        // 消息头
	data    []byte //消息数据
}

func NewHeader(code uint16, index uint16) *Header {
	return &Header{Code: code, Index: index}
}

func (this *Header) Size() int32 {
	return this.size
}

//Parse 解析二进制头并填充到对应字段
func (this *Header) Parse(head []byte) error {
	if len(head) != HeaderSize {
		return errors.New("head len error")
	}
	if err := utils.BytesToInt(head[0:4], &this.size); err != nil {
		return err
	}
	if err := utils.BytesToInt(head[4:6], &this.Code); err != nil {
		return err
	}

	if err := utils.BytesToInt(head[6:8], &this.Index); err != nil {
		return err
	}
	if this.size > Options.MaxDataSize {
		logger.Debug("包体太长，可能是包头错误:%+v", this)
		return ErrMsgDataSizeTooLong
	}
	return nil
}

//Bytes 生成二进制文件
func (this *Header) Bytes() (buffer *bytes.Buffer, err error) {
	buffer = new(bytes.Buffer)
	if err = utils.IntToBuffer(buffer, this.size); err != nil {
		return
	}
	if err = utils.IntToBuffer(buffer, this.Code); err != nil {
		return
	}
	if err = utils.IntToBuffer(buffer, this.Index); err != nil {
		return
	}
	return
}

func (this *Message) Read(p []byte) (n int, err error) {
	n = copy(p, this.data)
	return
}

func (this *Message) Write(p []byte) (n int, err error) {
	n = len(p)
	this.data = p
	this.size = int32(n)
	return
}

func (this *Message) Bytes() (b []byte, err error) {
	var buffer *bytes.Buffer
	buffer, err = this.Header.Bytes()
	if err != nil {
		return
	}
	if this.size > 0 {
		_, err = buffer.Write(this.data)
	}
	b = buffer.Bytes()
	return
}
