package cosnet

import (
	"github.com/hwcer/cosgo/binder"
	"testing"
)

func TestMessage(t *testing.T) {

	x := make([]byte, 10, 10)
	x = x[:0]
	t.Logf("b len:%v\n", len(x))
	t.Logf("b cap:%v\n", cap(x))
	t.Logf("b graw len:%v\n", len(x[0:10]))

	m := &Message{}
	_ = m.Marshal(-100, "path", "body", binder.Json)

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
