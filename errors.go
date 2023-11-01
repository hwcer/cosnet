package cosnet

import (
	"errors"
)

var (
	ErrAuthDataExist   = errors.New("authenticated")
	ErrAuthDataIllegal = errors.New("authentication data illegal")

	ErrSocketClosed      = errors.New("socket closed")
	ErrSocketChannelFull = errors.New("socket channel is full")
)
