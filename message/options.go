package message

import (
	"errors"
	"github.com/hwcer/cosgo/binder"
)

var ErrMsgHeadIllegal = errors.New("message head illegal")
var ErrMsgDataSizeTooLong = errors.New("message data too long")

var Options = struct {
	Binder      binder.Interface //UDP工作进程数量
	Capacity    int              //message []byte 默认长度
	MagicNumber byte
	MaxDataSize uint32
}{
	Binder:      binder.New(binder.MIMEJSON),
	Capacity:    1024,
	MagicNumber: 0x78,
	MaxDataSize: 1024 * 1024,
}
