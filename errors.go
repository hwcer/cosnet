package cosnet

import (
	"errors"
	"github.com/hwcer/cosgo/values"
)

var (
	ErrMsgDataSizeTooLong = errors.New("message data too long")

	ErrAuthDataExist   = errors.New("authenticated")
	ErrAuthDataIllegal = errors.New("authentication data illegal")

	ErrSocketClosed      = values.Errorf(0, "socket closed")
	ErrSocketChannelFull = values.Errorf(0, "socket channel is full")
)
