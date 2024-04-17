package message

import (
	"bytes"
	"testing"
)

type item struct {
	Id   int32
	Name string
}

func TestName(t *testing.T) {
	i := &item{Id: 1, Name: "s"}
	msg := &message{}
	if err := msg.Marshal("item", i); err != nil {
		t.Error(err)
	}
	t.Logf("Path:%v", string(msg.Path()))
	t.Logf("Body:%v", string(msg.Body()))

	buf := bytes.NewBuffer([]byte{})
	_, _ = msg.Bytes(buf)
	t.Logf("Bytes:%v", string(buf.String()))
}
