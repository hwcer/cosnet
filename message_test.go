package cosnet

import (
	"testing"
)

func TestMessage(t *testing.T) {
	m := &Message{}
	_ = m.Marshal(-100, "path", "body")

	b, _ := m.Bytes()
	t.Logf("M1:%+v", m)
	t.Logf("M1:%v", b)
	m2 := &Message{}
	if err := m2.Parse(b[0:MessageHead]); err != nil {
		t.Logf("%v", err)
	} else {
		t.Logf("%+v", m2)
	}

}
