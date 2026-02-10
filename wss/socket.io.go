package wss

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/hwcer/cosnet/listener"
	"github.com/hwcer/cosnet/message"
)

// SocketIO SOCKET.IO 协议的转换器
// 实现 transform 接口，用于处理 SOCKET.IO 协议的数据包
type SocketIO struct {
	Namespace string // 默认命名空间
}

// NewSocketIO 创建新的 SOCKET.IO 转换器
// 参数:
//
//	namespace: 默认命名空间，默认为 "/"
//
// 返回值:
//
//	SOCKET.IO 转换器实例
func NewSocketIO(namespace ...string) *SocketIO {
	t := &SocketIO{
		Namespace: "/",
	}
	if len(namespace) > 0 {
		t.Namespace = namespace[0]
	}
	return t
}

// ReadMessage 将 SOCKET.IO 格式的字节流转换为 cosnet 的消息对象
// 参数:
//
//	conn: WebSocket 连接对象
//	m: cosnet 消息对象
//	data: SOCKET.IO 格式的字节数据
//
// 返回值:
//
//	错误信息
func (t *SocketIO) ReadMessage(socket listener.Socket, m message.Message, data []byte) error {
	if len(data) == 0 {
		return errors.New("empty packet")
	}

	// 解析数据包类型
	packetType := data[0] - '0'

	switch packetType {
	case 0: // CONNECT
		// CONNECT 是底层包，跳过填充 Message
		return t.handleConnect(socket, m, data[1:])
	case 1: // DISCONNECT
		// DISCONNECT 是底层包，跳过填充 Message
		return t.handleDisconnect(socket, m, data[1:])
	case 2: // EVENT
		return t.handleEvent(socket, m, data[1:])
	case 3: // ACK
		// ACK 是底层包，跳过填充 Message
		return t.handleAck(socket, m, data[1:])
	case 5: // BINARY_EVENT
		return t.handleBinaryEvent(socket, m, data[1:])
	default:
		socket.Errorf("unsupported packet type: %d", packetType)
		return nil
	}
}

// WriteMessage 将 cosnet 的消息对象转换为 SOCKET.IO 格式的字节流
// 参数:
//
//	conn: WebSocket 连接对象
//	msg: cosnet 消息对象
//	b: 输出缓冲区
//
// 返回值:
//
//	写入的字节数和错误信息
func (t *SocketIO) WriteMessage(socket listener.Socket, msg message.Message, b *bytes.Buffer) (n int, err error) {
	// 获取消息路径（事件名称）
	path, _, err := msg.Path()
	if err != nil {
		return 0, err
	}

	// 获取消息数据
	body := msg.Body()

	flag := msg.Flag()
	// 获取消息索引
	index := msg.Index()

	var packet string
	if flag.Has(message.FlagIsACK) {
		// 当 index > 0 时，发送确认包（ACK）
		// 格式: "3/index,data"
		packet = fmt.Sprintf("3/%d,%s", index, string(body))
	} else {
		// 当 index <= 0 时，发送普通推送消息（EVENT）
		// 格式: "2/eventName,data"
		packet = fmt.Sprintf("2/%s,%s", path, string(body))
	}

	// 写入缓冲区
	n, err = b.WriteString(packet)
	if err != nil {
		return n, err
	}

	return n, nil
}

// handleConnect 处理 CONNECT 数据包
// 参数:
//
//	conn: WebSocket 连接对象
//	m: cosnet 消息对象
//	data: 数据包数据
//
// 返回值:
//
//	错误信息
func (t *SocketIO) handleConnect(socket listener.Socket, m message.Message, data []byte) error {
	// CONNECT 包格式: "0/namespace,[query]"
	// 例如: "0/chat,?token=abc123"

	// 解析命名空间
	ns := t.Namespace
	if len(data) > 0 {
		slashIdx := strings.IndexByte(string(data), '/')
		if slashIdx >= 0 {
			commaIdx := strings.IndexByte(string(data[slashIdx:]), ',')
			if commaIdx >= 0 {
				ns = string(data[slashIdx : slashIdx+commaIdx])
			} else {
				ns = string(data[slashIdx:])
			}
		}
	}

	// 生成会话ID
	sid := socket.Id()

	// 构建回复数据
	response := fmt.Sprintf(`{"sid": "%d", "upgrades": ["websocket"], "pingInterval": 25000, "pingTimeout": 5000}`, sid)

	// 构建 CONNECT 确认包
	connectPacket := fmt.Sprintf("0%s,%s", ns, response)

	conn := socket.Conn().(*Conn)
	// 发送回复包
	err := conn.Conn.WriteMessage(websocket.TextMessage, []byte(connectPacket))
	if err != nil {
		return err
	}

	return nil
}

// handleDisconnect 处理 DISCONNECT 数据包
// 参数:
//
//	conn: WebSocket 连接对象
//	m: cosnet 消息对象
//	data: 数据包数据
//
// 返回值:
//
//	错误信息
func (t *SocketIO) handleDisconnect(socket listener.Socket, m message.Message, data []byte) error {
	// DISCONNECT 包格式: "1/namespace"
	// 例如: "1/chat"

	// 这里可以触发断开连接事件
	// 具体实现取决于业务需求
	return net.ErrClosed
}

// handleEvent 处理 EVENT 数据包
// 参数:
//
//	conn: WebSocket 连接对象
//	m: cosnet 消息对象
//	data: 数据包数据
//
// 返回值:
//
//	错误信息
func (t *SocketIO) handleEvent(socket listener.Socket, m message.Message, data []byte) error {
	// EVENT 包格式: "2/eventName,data" 或 "2/eventName,ackId,[data1,data2,...]"
	// 例如: "2/chat,hello world" 或 "2/chat,123,["hello",123]"

	// 找到第一个逗号，分隔事件名称和数据
	commaIdx := strings.IndexByte(string(data), ',')
	if commaIdx == -1 {
		return errors.New("invalid event packet format: missing comma")
	}

	eventName := string(data[:commaIdx])
	remainingData := data[commaIdx+1:]

	// 检查是否包含确认ID
	var eventData []byte
	var ackId int64

	// 尝试解析确认ID（由上层处理）
	if len(remainingData) > 0 && remainingData[0] >= '0' && remainingData[0] <= '9' {
		// 查找确认ID后面的逗号
		ackCommaIdx := strings.IndexByte(string(remainingData), ',')
		if ackCommaIdx > 0 {
			// 解析确认ID
			ackIdStr := string(remainingData[:ackCommaIdx])
			var err error
			ackId, err = strconv.ParseInt(ackIdStr, 10, 64)
			if err == nil {
				// 确认ID解析成功，剩余部分是事件数据
				eventData = remainingData[ackCommaIdx+1:]
			} else {
				// 确认ID解析失败，整个剩余部分作为事件数据
				eventData = remainingData
			}
		} else {
			// 没有找到逗号，整个剩余部分作为事件数据
			eventData = remainingData
		}
	} else {
		// 不包含确认ID，整个剩余部分作为事件数据
		eventData = remainingData
	}
	var flag message.Flag
	if ackId > 0 {
		flag = message.FlagNeedACK
	}
	// 使用 MagicNumberPathJson 封装消息
	return m.Marshal(message.MagicNumberPathJson, flag, int32(ackId), eventName, eventData)
}

// handleAck 处理 ACK 数据包
// 参数:
//
//	conn: WebSocket 连接对象
//	m: cosnet 消息对象
//	data: 数据包数据
//
// 返回值:
//
//	错误信息
func (t *SocketIO) handleAck(socket listener.Socket, m message.Message, data []byte) error {
	// ACK 包格式: "3/ackId,data"
	// 例如: "3/123,success"

	// 解析确认 ID 和数据
	slashIdx := strings.IndexByte(string(data), '/')
	if slashIdx == -1 {
		return errors.New("invalid ack packet format: missing slash")
	}

	ackIdStr := string(data[:slashIdx])
	ackData := data[slashIdx+1:]

	// 解析确认 ID
	ackId, err := strconv.ParseInt(ackIdStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid ack id: %v", err)
	}

	// 这里可以处理确认响应
	// 具体实现取决于业务需求
	_ = ackId
	_ = ackData

	return nil
}

// handleBinaryEvent 处理 BINARY_EVENT 数据包
// 参数:
//
//	conn: WebSocket 连接对象
//	m: cosnet 消息对象
//	data: 数据包数据
//
// 返回值:
//
//	错误信息
func (t *SocketIO) handleBinaryEvent(socket listener.Socket, m message.Message, data []byte) error {
	// BINARY_EVENT 包格式: "5-attachments/eventName,data"
	// 例如: "5-1/chat,hello world"

	// 解析二进制附件数量
	dashIdx := strings.IndexByte(string(data), '-')
	if dashIdx == -1 {
		return errors.New("invalid binary event packet format: missing dash")
	}

	attachmentsStr := string(data[:dashIdx])
	attachments, err := strconv.Atoi(attachmentsStr)
	if err != nil {
		return fmt.Errorf("invalid attachments count: %v", err)
	}

	// 解析事件名称和数据
	remainingData := data[dashIdx+1:]
	commaIdx := strings.IndexByte(string(remainingData), ',')
	if commaIdx == -1 {
		return errors.New("invalid binary event packet format: missing comma")
	}

	eventName := string(remainingData[:commaIdx])
	remainingEventData := remainingData[commaIdx+1:]

	// 检查是否包含确认ID
	var eventData []byte
	var ackId int64

	// 尝试解析确认ID（由上层处理）
	if len(remainingEventData) > 0 && remainingEventData[0] >= '0' && remainingEventData[0] <= '9' {
		// 查找确认ID后面的逗号
		ackCommaIdx := strings.IndexByte(string(remainingEventData), ',')
		if ackCommaIdx > 0 {
			// 解析确认ID
			ackIdStr := string(remainingEventData[:ackCommaIdx])
			ackId, err = strconv.ParseInt(ackIdStr, 10, 64)
			if err == nil {
				// 确认ID解析成功，剩余部分是事件数据
				eventData = remainingEventData[ackCommaIdx+1:]
			} else {
				// 确认ID解析失败，整个剩余部分作为事件数据
				eventData = remainingEventData
			}
		} else {
			// 没有找到逗号，整个剩余部分作为事件数据
			eventData = remainingEventData
		}
	} else {
		// 不包含确认ID，整个剩余部分作为事件数据
		eventData = remainingEventData
	}

	// 使用 MagicNumberPathJson 封装消息
	// 参考 socket.go 中的 Send 方法

	var flag message.Flag
	if ackId > 0 {
		flag = message.FlagNeedACK
	}
	if err = m.Marshal(message.MagicNumberPathJson, flag, int32(ackId), eventName, eventData); err != nil {
		return err
	}

	// 注意：这里需要处理二进制附件
	// 具体实现取决于业务需求
	_ = attachments

	return nil
}

// BuildConnectPacket 构建 CONNECT 数据包
// 参数:
//
//	namespace: 命名空间
//
// 返回值:
//
//	CONNECT 数据包的字节数组
func (t *SocketIO) BuildConnectPacket(namespace ...string) []byte {
	ns := t.Namespace
	if len(namespace) > 0 {
		ns = namespace[0]
	}
	return []byte(fmt.Sprintf("0/%s", ns))
}

// BuildDisconnectPacket 构建 DISCONNECT 数据包
// 参数:
//
//	namespace: 命名空间
//
// 返回值:
//
//	DISCONNECT 数据包的字节数组
func (t *SocketIO) BuildDisconnectPacket(namespace ...string) []byte {
	ns := t.Namespace
	if len(namespace) > 0 {
		ns = namespace[0]
	}
	return []byte(fmt.Sprintf("1/%s", ns))
}

// BuildEventPacket 构建 EVENT 数据包
// 参数:
//
//	eventName: 事件名称
//	data: 事件数据
//	ackId: 确认ID（可选，0表示不需要确认）
//
// 返回值:
//
//	EVENT 数据包的字节数组
func (t *SocketIO) BuildEventPacket(eventName string, data []byte, ackId ...int64) []byte {
	if len(ackId) > 0 && ackId[0] > 0 {
		return []byte(fmt.Sprintf("2/%s,%d,%s", eventName, ackId[0], string(data)))
	}
	return []byte(fmt.Sprintf("2/%s,%s", eventName, string(data)))
}

// BuildAckPacket 构建 ACK 数据包
// 参数:
//
//	ackId: 确认 ID
//	data: 确认数据
//
// 返回值:
//
//	ACK 数据包的字节数组
func (t *SocketIO) BuildAckPacket(ackId int64, data []byte) []byte {
	return []byte(fmt.Sprintf("3/%d,%s", ackId, string(data)))
}

// BuildBinaryEventPacket 构建 BINARY_EVENT 数据包
// 参数:
//
//	attachments: 二进制附件数量
//	eventName: 事件名称
//	data: 事件数据
//	ackId: 确认ID（可选，0表示不需要确认）
//
// 返回值:
//
//	BINARY_EVENT 数据包的字节数组
func (t *SocketIO) BuildBinaryEventPacket(attachments int, eventName string, data []byte, ackId ...int64) []byte {
	if len(ackId) > 0 && ackId[0] > 0 {
		return []byte(fmt.Sprintf("5-%d/%s,%d,%s", attachments, eventName, ackId[0], string(data)))
	}
	return []byte(fmt.Sprintf("5-%d/%s,%s", attachments, eventName, string(data)))
}
