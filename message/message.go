package message

import (
	"bytes"
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
		r = string(m.bytes[4 : int(code)+4])
		if i := strings.Index(r, "?"); i >= 0 {
			r = r[:i]
			q = r[i+1:]
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
	if includeHeader {
		head := m.Head.bytes()
		if r, err = w.Write(head); err == nil {
			n += r
		} else {
			return
		}
	}
	if size > 0 {
		r, err = w.Write(m.bytes[0:size])
		n += r
	}
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
	return nil
}

// Marshal 将一个对象放入Message.data
func (m *message) Marshal(magic byte, index int32, path string, body any) (err error) {
	if err = m.Head.format(magic, index); err != nil {
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

func (m *message) Confirm() (string, bool) {
	magic := m.Magic()
	if !magic.Confirm {
		return "", false
	}
	var p string
	//如果包序号为0时原路返回
	if m.Index() > 0 && Options.S2CConfirm != "" {
		p = Options.S2CConfirm
	} else {
		p, _, _ = m.Path()
	}
	return p, true
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
