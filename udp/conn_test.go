package udp

import (
	"testing"
	"unsafe"
)

func TestName(t *testing.T) {
	var c *Conn

	t.Logf("p :%v", unsafe.Sizeof(c))

	t.Logf("p.addr :%v", unsafe.Sizeof(c.addr))
	t.Logf("p.Mutex :%v", unsafe.Sizeof(c.Mutex))

}
