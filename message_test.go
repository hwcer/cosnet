package cosnet

import "testing"

func TestMessage(t *testing.T) {
	m := &Message{code: -100, path: 0, body: 0}
	b, _ := m.Bytes()
	t.Logf("M1:%+v", m)
	t.Logf("M1:%+v", m)
	m2 := &Message{}
	if err := m2.Parse(b); err != nil {
		t.Logf("%v", err)
	} else {
		t.Logf("%+v", m2)
	}
}
