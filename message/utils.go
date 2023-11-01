package message

import (
	"github.com/soheilhy/cmux"
	"io"
)

type Message interface {
	Path() string
	Body() []byte
	Parse(head []byte) error
	Bytes(w io.Writer) (n int, err error)
	Write(r io.Reader) (n int, err error)
	Marshal(path string, body any) error
	Unmarshal(i interface{}) (err error)
}

type Handler interface {
	Head() []byte
	Require() Message
	Release(Message)
}

/*
Matcher cmux Matcher
m := cmux.New(ln)
ln := m.Match(Matcher())
Server.TCPListener(ln)
*/
func Matcher() cmux.Matcher {
	magic := Options.MagicNumber
	return func(r io.Reader) bool {
		buf := make([]byte, 1)
		n, _ := r.Read(buf)
		return n == 1 && buf[0] == magic
	}
}
