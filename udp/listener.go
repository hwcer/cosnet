package udp

import (
	"errors"
	"net"
	"sync"

	"github.com/hwcer/cosgo/scc"
	"github.com/hwcer/cosnet/listener"
	"github.com/hwcer/logger"
)

// Listener 实现listener.Listener接口
type Listener struct {
	ln      *net.UDPConn
	connCh  chan *Conn
	addr    net.Addr
	network string
	conns   map[string]*Conn // 用于跟踪活跃的Conn对象
	mu      sync.Mutex       // 用于保护conns map的并发访问
}

// New 创建一个新的udp监听器
func New(network, address string) (listener.Listener, error) {
	addr, err := net.ResolveUDPAddr(network, address)
	if err != nil {
		return nil, err
	}
	ln, err := net.ListenUDP(network, addr)
	if err != nil {
		return nil, err
	}
	result := &Listener{
		ln:      ln,
		connCh:  make(chan *Conn, Options.ConnChanSize), // 使用配置的通道大小
		addr:    ln.LocalAddr(),
		network: network,
		conns:   make(map[string]*Conn),
	}
	scc.GO(result.readLoop)
	return result, nil
}

// Accept 等待并返回下一个连接到监听器
func (ln *Listener) Accept() (listener.Conn, error) {
	conn, ok := <-ln.connCh
	if !ok {
		return nil, errors.New("udp listener closed")
	}
	return conn, nil
}

// Close 关闭监听器
func (ln *Listener) Close() error {
	close(ln.connCh)
	// 关闭所有活跃的Conn对象，使用写锁
	ln.mu.Lock()
	for _, conn := range ln.conns {
		conn.Close()
	}
	ln.conns = nil
	ln.mu.Unlock()
	return ln.ln.Close()
}

// Addr 返回监听器的网络地址
func (ln *Listener) Addr() net.Addr {
	return ln.addr
}

// newConn 创建一个新的UDP连接
// 注意：调用此方法前，必须已经持有ln.mu互斥锁
func (ln *Listener) newConn(conn *net.UDPConn, addr *net.UDPAddr, key string) (r *Conn, err error) {
	r = &Conn{
		conn:    conn,
		addr:    addr,
		msgChan: make(chan []byte, Options.MsgChanSize), // 使用配置的通道大小
		ln:      ln,
		key:     key,
	}
	// 发送到通道
	select {
	case ln.connCh <- r:
	default:
		return nil, errors.New("udp listener conn channel full, drop conn")
		// 通道已满，丢弃该连接
	}
	// 直接添加到conns map，因为调用者已经持有了互斥锁
	ln.conns[key] = r
	return r, nil
}

// readLoop 读取UDP数据包并创建连接
func (ln *Listener) readLoop() {
	buffer := make([]byte, 65535) // UDP最大数据包大小
	for !scc.Stopped() {
		n, addr, err := ln.ln.ReadFromUDP(buffer)
		if err != nil {
			break
		}
		if n > 0 {
			// 生成端点的唯一标识
			addrKey := addr.String()
			// 检查是否已存在该端点的Conn对象
			ln.mu.Lock()
			conn, exists := ln.conns[addrKey]
			if !exists {
				// 创建一个新的UDP连接
				conn, err = ln.newConn(ln.ln, addr, addrKey)
				if err != nil {
					ln.mu.Unlock()
					continue
				}
			}
			ln.mu.Unlock()

			// 复制数据到新的切片，避免缓冲区被覆盖
			data := make([]byte, n)
			copy(data, buffer[:n])

			// 将数据包发送到Conn对象的msgChan中
			select {
			case conn.msgChan <- data:
			default:
				logger.Alert("udp conn %s msg channel full, drop msg", conn.key)
				// 通道已满，丢弃该数据包
			}
		}
	}
}

// removeConn 从活跃连接列表中移除Conn对象
func (ln *Listener) removeConn(key string) {
	// 移除操作需要写锁
	ln.mu.Lock()
	delete(ln.conns, key)
	ln.mu.Unlock()
}
