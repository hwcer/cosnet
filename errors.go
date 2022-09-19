package cosnet

import "errors"

var (
	ErrMsgDataSizeTooLong = errors.New("Message data too long")

	ErrAuthDataExist   = errors.New("authenticated")
	ErrAuthDataIllegal = errors.New("authentication data illegal")
)
