package message

import "testing"

func TestName(t *testing.T) {
	b := make([]byte, 5, 10)
	c := b[0:11]
	t.Logf("len:%v    cap:%v", len(c), cap(c))

}
