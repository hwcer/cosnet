package cosnet

import "io"

// iHandle 控制器
type iHandle interface {
	New() iMessage //NEW MESSAGE
	Size() int     //包头长度
}

type iMessage interface {
	Len() int
	Path() string
	Parse(head []byte) error
	Bytes(w io.Writer) (n int, err error)
	Write(r io.Reader) (n int, err error)
}
