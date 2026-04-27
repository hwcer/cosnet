package message

import (
	"testing"
)

// TestPoolRequireRelease 验证消息池 Require/Release 循环正确性
func TestPoolRequireRelease(t *testing.T) {
	Options.Pool = true
	m := Require()
	if m == nil {
		t.Fatal("Require returned nil")
	}

	// 模拟使用：填充 head
	err := m.Marshal(MagicNumberPathJson, 0, 1, "/test", []byte("hello"))
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Release 后字段应被重置
	Release(m)

	// 再次 Require，应拿到重置后的 message（Head 已清零，不调用 Code 因为 Magic=0）
	m2 := Require()
	if m2.Size() != 0 {
		t.Errorf("pooled message Size not reset: %d", m2.Size())
	}
	if m2.Flag() != 0 {
		t.Errorf("pooled message Flag not reset: %d", m2.Flag())
	}
	if m2.Index() != 0 {
		t.Errorf("pooled message Index not reset: %d", m2.Index())
	}
	Release(m2)
}

// TestPoolDisabled 验证 Pool=false 时 Require/Release 不 panic
func TestPoolDisabled(t *testing.T) {
	Options.Pool = false
	m := Require()
	if m == nil {
		t.Fatal("Require returned nil with pool disabled")
	}
	Release(m)  // 应为 no-op，不 panic
	Release(nil) // nil 安全
	Options.Pool = true
}

// TestMarshalUnmarshalPath 验证 path 模式消息编解码往返
func TestMarshalUnmarshalPath(t *testing.T) {
	m := Require()
	defer Release(m)

	body := []byte(`{"name":"test"}`)
	err := m.Marshal(MagicNumberPathJson, 0, 42, "/api/echo", body)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	path, query, err := m.Path()
	if err != nil {
		t.Fatalf("Path error: %v", err)
	}
	if path != "/api/echo" {
		t.Errorf("Path mismatch: got %q, want /api/echo", path)
	}
	if query != "" {
		t.Errorf("Query should be empty, got %q", query)
	}
	if m.Index() != 42 {
		t.Errorf("Index mismatch: got %d, want 42", m.Index())
	}

	gotBody := m.Body()
	if string(gotBody) != string(body) {
		t.Errorf("Body mismatch: got %q, want %q", gotBody, body)
	}
}

// TestMarshalPathWithQuery 验证带查询参数的 path 解析
func TestMarshalPathWithQuery(t *testing.T) {
	m := Require()
	defer Release(m)

	err := m.Marshal(MagicNumberPathJson, 0, 0, "/chat?room=1&user=abc", []byte("hi"))
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	path, query, err := m.Path()
	if err != nil {
		t.Fatalf("Path error: %v", err)
	}
	if path != "/chat" {
		t.Errorf("Path: got %q, want /chat", path)
	}
	if query != "room=1&user=abc" {
		t.Errorf("Query: got %q, want room=1&user=abc", query)
	}
}

// TestPathNegativeCode 验证恶意负数 code 不会 panic（崩服修复验证）
func TestPathNegativeCode(t *testing.T) {
	m := &message{}
	// 手动构造恶意 head：magic=PathJson, size=4
	head := make([]byte, messageHeadSize)
	head[0] = MagicNumberPathJson
	head[1] = 0
	magic := Magics.Get(MagicNumberPathJson)
	magic.Binary.PutUint32(head[2:6], 4)  // size=4
	magic.Binary.PutUint32(head[6:10], 0) // index=0

	if err := m.Head.Parse(head); err != nil {
		t.Fatalf("Parse head error: %v", err)
	}

	// 设置 body：4 字节的 0xFFFFFFFF（path length = -1 as int32）
	m.bytes = []byte{0xFF, 0xFF, 0xFF, 0xFF}

	// Path() 应返回 error，不应 panic
	_, _, err := m.Path()
	if err == nil {
		t.Error("expected error for negative code path, got nil")
	}

	// Body() 应返回 nil 或安全值，不应 panic
	body := m.Body()
	if body == nil {
		// OK: nil is safe
	}
	_ = body
}

// TestPathZeroCode 验证 code=0 的边界情况
func TestPathZeroCode(t *testing.T) {
	m := &message{}
	head := make([]byte, messageHeadSize)
	head[0] = MagicNumberPathJson
	magic := Magics.Get(MagicNumberPathJson)
	magic.Binary.PutUint32(head[2:6], 4)
	m.Head.Parse(head)

	// code=0 表示 path 长度为 0
	m.bytes = []byte{0x00, 0x00, 0x00, 0x00}

	_, _, err := m.Path()
	if err == nil {
		t.Error("expected error for zero code path, got nil")
	}
}

// TestFlagOperations 验证 Flag 位操作
func TestFlagOperations(t *testing.T) {
	var f Flag

	f.Set(FlagConfirm)
	if !f.Has(FlagConfirm) {
		t.Error("FlagConfirm not set")
	}

	f.Set(FlagHeartbeat)
	if !f.Has(FlagHeartbeat) {
		t.Error("FlagHeartbeat not set")
	}
	if !f.Has(FlagConfirm) {
		t.Error("FlagConfirm lost after setting FlagHeartbeat")
	}

	f.Delete(FlagConfirm)
	if f.Has(FlagConfirm) {
		t.Error("FlagConfirm not deleted")
	}
	if !f.Has(FlagHeartbeat) {
		t.Error("FlagHeartbeat lost after deleting FlagConfirm")
	}
}

// TestMagicRegistry 验证 Magic 注册和查找
func TestMagicRegistry(t *testing.T) {
	if !Magics.Has(MagicNumberPathJson) {
		t.Error("MagicNumberPathJson not registered")
	}
	if !Magics.Has(MagicNumberCodeJson) {
		t.Error("MagicNumberCodeJson not registered")
	}
	if !Magics.Has(MagicNumberCodeProto) {
		t.Error("MagicNumberCodeProto not registered")
	}
	if Magics.Has(0x00) {
		t.Error("0x00 should not be registered")
	}

	m := Magics.Get(MagicNumberPathJson)
	if m == nil {
		t.Fatal("Get MagicNumberPathJson returned nil")
	}
	if m.Type != MagicTypePath {
		t.Errorf("MagicNumberPathJson type: got %d, want %d", m.Type, MagicTypePath)
	}
}
