package message

import (
	"bytes"
	"testing"
)

type foo struct {
	Name string
}

func TestName(t *testing.T) {
	msg := &message{}
	d := &foo{Name: "test"}
	if err := msg.Marshal(MagicNumberPathJsonWithConfirm, 101, "test", d); err != nil {
		t.Error(err)
		return
	}

	buf := bytes.NewBuffer(make([]byte, 0))
	if _, err := msg.Bytes(buf, true); err != nil {
		t.Error(err)
		return
	}

	t.Log(buf.String())

	d2 := &foo{}
	m2 := &message{}
	if err := m2.Reset(buf.Bytes()); err != nil {
		t.Error(err)
		return
	}

	if err := m2.Unmarshal(d2); err != nil {
		t.Error(err)
	}
	t.Log(d2)

}
