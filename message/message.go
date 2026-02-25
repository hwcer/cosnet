package message

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"strings"
)

type message struct {
	Head
	code  int32
	bytes []byte //数据   uint32 (path) body
}

func (m *message) Code() int32 {
	if m.code == 0 {
		magic := m.Magic()
		m.code = int32(magic.Binary.Uint32(m.bytes[0:4]))
	}
	return m.code
}
func (m *message) Path() (r, q string, err error) {
	magic := m.Magic()
	code := m.Code()
	if magic.Type == MagicTypePath {
		if int(code) > len(m.bytes) {
			err = ErrMsgHeadIllegal
			return
		}
		path := string(m.bytes[4 : int(code)+4])
		if i := strings.Index(path, "?"); i >= 0 {
			r = path[:i]
			q = path[i+1:]
		} else {
			r = path
		}
	} else {
		r, err = Transform.Path(code)
	}
	return
}

// Body 消息体
func (m *message) Body() []byte {
	magic := m.Magic()
	i := int(4)
	if magic.Type == MagicTypePath {
		i += int(m.Code())
	}
	return m.bytes[i:]
}

// Bytes 生成二进制文件
func (m *message) Bytes(w io.Writer, includeHeader bool) (n int, err error) {
	var r int
	size := m.Size()
	compressed := false
	if includeHeader {
		var head []byte
		head, compressed = m.Head.bytes()
		if r, err = w.Write(head); err != nil {
			return
		}
		n += r
	} else {
		compressed = Options.AutoCompressSize > 0 && size > Options.AutoCompressSize && !m.Head.flag.Has(FlagCompressed)
	}
	if size == 0 {
		return
	}
	// 写入数据体
	if compressed {
		r, err = m.compress(w)
	} else {
		r, err = w.Write(m.bytes)
	}
	n += r
	return
}

// Write 从conn中读取数据写入到data
func (m *message) Write(r io.Reader) (n int, err error) {
	size := int(m.Size())
	if size == 0 {
		return
	}
	if cap(m.bytes) >= size {
		m.bytes = m.bytes[0:size]
	} else if size > Options.Capacity {
		m.bytes = make([]byte, size)
	} else {
		m.bytes = make([]byte, size, Options.Capacity)
	}
	n, err = io.ReadFull(r, m.bytes[0:size])
	if n != size {
		return n, io.ErrShortBuffer
	}
	// 解压数据（如果已压缩）
	if err = m.decompress(); err != nil {
		return n, err
	}
	return
}

// Reset  WS UDP 数据包模式直接填充
func (m *message) Reset(b []byte) error {
	if len(b) < messageHeadSize {
		return io.ErrUnexpectedEOF
	}
	if err := m.Head.Parse(b[0:messageHeadSize]); err != nil {
		return err
	}
	m.bytes = b[messageHeadSize:]
	// 解压数据（如果已压缩）
	return m.decompress()
}

// Marshal 将一个对象放入Message.data
func (m *message) Marshal(magic byte, flag Flag, index int32, path string, body any) (err error) {
	if err = m.Head.format(magic, flag, index); err != nil {
		return
	}
	mc := m.Magic()
	if len(m.bytes) < 4 {
		m.bytes = make([]byte, 4, Options.Capacity)
	}

	var buffer *bytes.Buffer
	if mc.Type == MagicTypePath {
		mc.Binary.PutUint32(m.bytes[0:4], uint32(len(path)))
		buffer = bytes.NewBuffer(m.bytes[0:4])
		buffer.WriteString(path)
	} else {
		var code int32
		if code, err = Transform.Code(path); err == nil {
			return err
		}
		mc.Binary.PutUint32(m.bytes[0:4], uint32(code))
		buffer = bytes.NewBuffer(m.bytes[0:4])
	}

	switch v := body.(type) {
	case []byte:
		if len(v) > 0 {
			buffer.Write(v)
		}
	default:
		bi := m.Binder()
		if bi == nil {
			err = errors.New("no binder found")
		} else {
			err = bi.Encode(buffer, body)
		}
	}
	if err != nil {
		return err
	}
	m.bytes = buffer.Bytes()
	m.size = int32(len(m.bytes))
	return nil
}

// Unmarshal 解析Message body
func (m *message) Unmarshal(i any) (err error) {
	bi := m.Binder()
	if bi == nil {
		return errors.New("no binder found")
	}
	return bi.Unmarshal(m.Body(), i)
}

// decompress 解压 gzip 压缩的数据，如果未压缩则直接返回
func (m *message) decompress() error {
	if !m.Head.flag.Has(FlagCompressed) {
		return nil
	}
	gr, err := gzip.NewReader(bytes.NewReader(m.bytes))
	if err != nil {
		return err
	}
	defer gr.Close()
	decompressed, err := io.ReadAll(gr)
	if err != nil {
		return err
	}
	m.bytes = decompressed
	m.size = int32(len(m.bytes))
	m.Head.flag.Delete(FlagCompressed)
	return nil
}

// compress 压缩数据并直接写入 writer
// 注意：此方法不修改 m.bytes，保持原始数据未压缩状态
func (m *message) compress(w io.Writer) (int, error) {
	gw := gzip.NewWriter(w)
	n, err := gw.Write(m.bytes)
	if err != nil {
		return n, err
	}
	if err := gw.Close(); err != nil {
		return n, err
	}
	return n, nil
}

func (m *message) Confirm() string {
	var p string
	if Options.S2CConfirm != "" {
		p = Options.S2CConfirm
	} else {
		p, _, _ = m.Path()
	}
	return p
}
func (m *message) Release() {
	m.Head.Release()
	m.code = 0
	// 重置 bytes 字段，避免内存泄漏和数据污染
	if cap(m.bytes) > Options.Capacity {
		// 如果容量过大，创建新的切片
		m.bytes = make([]byte, 0, Options.Capacity)
	} else {
		// 否则重置切片长度
		m.bytes = m.bytes[:0]
	}
}
