package message

import (
	"errors"
	"io"

	"github.com/hwcer/cosgo/binder"
)

var ErrMsgHeadIllegal = errors.New("message head illegal")
var ErrMsgDataSizeTooLong = errors.New("message data too long")
var ErrMsgHeadNotSetTransform = errors.New("code mode, please set the message transform first")

var Options = struct {
	Pool             bool //是否启用消息池 message pool
	Capacity         int  //message []byte 默认长度
	MaxDataSize      int32
	AutoCompressSize int32  //自动压缩的阈值，超过此大小的消息会被自动压缩, 0 表示不压缩
	S2CConfirm       string //确认包协议，默认原路返回(和请求时一致)
	New              func() Message
	Head             func() []byte //包头
}{
	Pool:             true,
	Capacity:         1024,
	MaxDataSize:      1024 * 1024,
	AutoCompressSize: 1024 * 100, //超过 100KB 自动压缩
	New:              func() Message { return &message{} },
	Head:             func() []byte { return make([]byte, messageHeadSize) },
}

type Message interface {
	Flag() Flag                                                              //标签
	Size() int32                                                             //包体长度
	Code() int32                                                             //协议号，或者path长度
	Path() (r string, q string, err error)                                   //转发路径，数字类型的协议号需要转换成 /servicePath/servicesMethod
	Index() int32                                                            //自增
	Magic() *Magic                                                           //获取魔法设定参数
	Body() []byte                                                            //二进制包体
	Reset([]byte) error                                                      //使用完整二进制重置MESSAGE,websocket,udp模式
	Parse(head []byte) error                                                 //解析二进制包头
	Bytes(w io.Writer, head bool) (n int, err error)                         //转换成二进制并发送
	Write(r io.Reader) (n int, err error)                                    //从CONN中写入Size()字节
	Binder() binder.Binder                                                   //当前协议使用的系列化方式
	Marshal(magic byte, flag Flag, index int32, path string, body any) error //使用对象填充包体
	Unmarshal(i any) (err error)                                             //解析包体
	Confirm() string                                                         //确认包路径
	Release()
}
