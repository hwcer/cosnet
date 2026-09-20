package message

import (
	"bytes"
	"compress/gzip"
	"testing"
)

// 🔴 回归:解压超限必须拒绝而非静默截断——LimitReader 恰好在 limit 处返回 EOF,
// 直接 ReadAll(limit) 会把超限炸弹截成 limit 字节的损坏消息正常投递
func TestDecompressRejectsOversizedPayload(t *testing.T) {
	old := Options.MaxDataSize
	Options.MaxDataSize = 64
	defer func() { Options.MaxDataSize = old }()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	//膨胀 64*4 上限数倍的数据
	payload := bytes.Repeat([]byte("A"), 64*4*3)
	if _, err := gw.Write(payload); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}

	m := &message{bytes: buf.Bytes()}
	m.Head.flag.Set(FlagCompressed)
	if err := m.decompress(); err != ErrMsgDataSizeTooLong {
		t.Fatalf("超限包应返回 ErrMsgDataSizeTooLong,实际:%v", err)
	}
}

// 上限内的正常压缩包照常解压
func TestDecompressWithinLimit(t *testing.T) {
	old := Options.MaxDataSize
	Options.MaxDataSize = 64
	defer func() { Options.MaxDataSize = old }()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	payload := bytes.Repeat([]byte("A"), 100)
	if _, err := gw.Write(payload); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}

	m := &message{bytes: buf.Bytes()}
	m.Head.flag.Set(FlagCompressed)
	if err := m.decompress(); err != nil {
		t.Fatalf("上限内应解压成功:%v", err)
	}
	if !bytes.Equal(m.bytes, payload) {
		t.Fatalf("解压结果不符:%d bytes", len(m.bytes))
	}
}
