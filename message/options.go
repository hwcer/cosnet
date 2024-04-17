package message

import (
	"errors"
	"github.com/hwcer/cosgo/binder"
	"io"
)

var ErrMsgHeadIllegal = errors.New("message head illegal")
var ErrMsgDataSizeTooLong = errors.New("message data too long")

var Options = struct {
	Binder      binder.Interface //UDP工作进程数量
	Capacity    int              //message []byte 默认长度
	MagicNumber byte
	MaxDataSize int32
	New         func() Message
	Head        func() []byte //包头
}{
	Binder:      binder.New(binder.MIMEJSON),
	Capacity:    1024,
	MagicNumber: 0x78,
	MaxDataSize: 1024 * 1024,
	New:         func() Message { return &message{} },
	Head:        func() []byte { return make([]byte, messageHeadSize) },
}

type Message interface {
	Size() int32
	Path() string
	Body() []byte
	Reset([]byte)                         //使用完成二进制包体重置MESSAGE
	Parse(head []byte) error              //解析二进制包头
	Bytes(w io.Writer) (n int, err error) //转换成二进制并发送
	Write(r io.Reader) (n int, err error) //从CONN中写入Size()哥字节
	Marshal(path string, body any) error  //使用对象填充包体
	Unmarshal(i interface{}) (err error)
	Release()
}
