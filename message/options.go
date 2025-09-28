package message

import (
	"errors"
	"io"
)

var ErrMsgHeadIllegal = errors.New("message head illegal")
var ErrMsgDataSizeTooLong = errors.New("message data too long")
var ErrMsgHeadNotSetTransform = errors.New("code mode, please set the message transform first")

var Options = struct {
	Pool        bool //是否启用消息池 message pool
	Capacity    int  //message []byte 默认长度
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
	Size() uint32                                                  //包体长度
	Code() int32                                                   //协议号，或者path长度
	Path() (r string, q string, err error)                         //转发路径，数字类型的协议号需要转换成 /servicePath/servicesMethod
	Index() uint32                                                 //自增
	Magic() *Magic                                                 //获取魔法设定参数
	Body() []byte                                                  //二进制包体
	Reset([]byte) error                                            //使用完整二进制重置MESSAGE,websocket,udp模式
	Parse(head []byte) error                                       //解析二进制包头
	Bytes(w io.Writer, head bool) (n int, err error)               //转换成二进制并发送
	Write(r io.Reader) (n int, err error)                          //从CONN中写入Size()字节
	Marshal(magic byte, index uint32, path string, body any) error //使用对象填充包体
	Unmarshal(i any) (err error)
	Release()
	Confirm() bool //是否需要回复确认包
}
