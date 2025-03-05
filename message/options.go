package message

import (
	"errors"
	"io"
)

const (
	HeadModePath uint8 = 0
	HeadModeCode uint8 = 1
)

const (
	MagicNumber  byte = 0x77
	MagicConfirm byte = 0x78 //需要应答
)

var ErrMsgHeadIllegal = errors.New("message head illegal")
var ErrMsgDataSizeTooLong = errors.New("message data too long")
var ErrMsgHeadNotSetTransform = errors.New("code mode, please set the message transform first")

var Options = struct {
	Mode        uint8 //工作模式:0-path, 1-code
	Pool        bool  //是否启用消息池 message pool
	Capacity    int   //message []byte 默认长度
	MaxDataSize uint32
	New         func() Message
	Head        func() []byte //包头
}{
	Pool:        true,
	Capacity:    1024,
	MaxDataSize: 1024 * 1024,
	New:         func() Message { return &message{} },
	Head:        func() []byte { return make([]byte, messageHeadSize) },
}

type Message interface {
	Size() uint32                                                //包体长度
	Path() (string, error)                                       //转发路径，数字类型的协议号需要转换成 /servicePath/servicesMethod
	Query() map[string]string                                    //头部参数
	Body() []byte                                                //包头二进制
	Reset([]byte) error                                          //使用完整二进制重置MESSAGE,websocket udp	模式
	Parse(head []byte) error                                     //解析二进制包头
	Bytes(w io.Writer, head bool) (n int, err error)             //转换成二进制并发送
	Write(r io.Reader) (n int, err error)                        //从CONN中写入Size()字节
	Verify() error                                               //校验包体是否正常
	Marshal(path string, body any, meta map[string]string) error //使用对象填充包体
	Unmarshal(i any) (err error)
	Release()
	Confirm() bool //是否需要回复确认包
}
