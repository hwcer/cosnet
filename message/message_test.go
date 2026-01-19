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
	if err := msg.Marshal(MagicNumberPathJsonConfirm, 101, "test", d); err != nil {
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

// TestMessagePoolRecycling 测试消息池回收情况
func TestMessagePoolRecycling(t *testing.T) {
	// 确保消息池启用
	oldPool := Options.Pool
	Options.Pool = true
	defer func() {
		Options.Pool = oldPool
	}()

	// 获取第一个消息对象
	msg1 := Require()
	d1 := &foo{Name: "test1"}
	if err := msg1.Marshal(MagicNumberPathJsonConfirm, 101, "test", d1); err != nil {
		t.Error(err)
		return
	}

	// 记录 msg1 的地址
	msg1Addr := &msg1

	// 释放 msg1
	Release(msg1)

	// 获取第二个消息对象（应该是从池中复用的）
	msg2 := Require()
	msg2Addr := &msg2

	// 检查是否复用了相同的对象
	t.Logf("msg1 address: %p, msg2 address: %p", msg1Addr, msg2Addr)

	// 验证 msg2 是否被正确重置
	// 检查 Magic 字段
	if msg2.Magic() != nil {
		t.Error("msg2.Magic() should be nil after recycling")
	}

	// 检查 Size 字段
	if msg2.Size() != 0 {
		t.Error("msg2.Size() should be 0 after recycling")
	}

	// 检查 Index 字段
	if msg2.Index() != 0 {
		t.Error("msg2.Index() should be 0 after recycling")
	}

	// 释放 msg2
	Release(msg2)
}
