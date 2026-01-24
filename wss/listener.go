package wss

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hwcer/cosgo/scc"
	"github.com/hwcer/cosnet/listener"
	"github.com/hwcer/logger"
)

// New 创建一个新的wss监听器
// 仅仅提供给cosnet快速创建 wss 服务器
// network : "ws", "wss", "wss4", "wss5", "wss6"
func New(network, address string, tlsConfig ...*tls.Config) (listener.Listener, error) {
	srv := &http.Server{
		Addr:              address,
		ReadHeaderTimeout: 3 * time.Second,
		//Handler:           h,
	}

	if len(tlsConfig) > 0 {
		srv.TLSConfig = tlsConfig[0]
	} else if network == "wss" || network == "wss4" || network == "wss5" || network == "wss6" {
		// 如果需要TLS，确保提供了TLS配置
		return nil, errors.New("TLS configuration is required for wss network type")
	}

	ln := NewListener(srv, "")
	//启动服务
	err := scc.Timeout(time.Second, func() error {
		if srv.TLSConfig != nil {
			return srv.ListenAndServeTLS("", "")
		}
		return srv.ListenAndServe()
	})
	if errors.Is(err, scc.ErrorTimeout) {
		err = nil
	}
	return ln, err
}

func NewListener(srv *http.Server, route string) *Listener {
	ln := &Listener{
		route:    route,
		server:   srv,
		connChan: make(chan *websocket.Conn, Options.ConnChanSize), // 使用配置的通道大小
	}
	srv.Handler = ln
	return ln
}

// Listener 实现listener.Listener接口
type Listener struct {
	err      error
	route    string
	server   *http.Server
	connChan chan *websocket.Conn
}

// Accept 等待并返回下一个连接到监听器
func (ln *Listener) Accept() (listener.Conn, error) {
	if ln.err != nil {
		return nil, ln.err
	}
	conn := <-ln.connChan
	wssConn := NewConn(conn)
	return wssConn, nil
}

// Close 关闭监听器
func (ln *Listener) Close() error {
	return ln.server.Close()
}

// Addr 返回监听器的网络地址
func (ln *Listener) Addr() net.Addr {
	// 尝试解析http.Server的Addr字段，返回一个net.Addr类型的值
	addr, err := net.ResolveTCPAddr("tcp", ln.server.Addr)
	if err != nil {
		// 如果解析失败，返回一个空的TCPAddr
		return &net.TCPAddr{}
	}
	return addr
}

func (ln *Listener) HTTPErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(500)
	if r.Method != http.MethodHead {
		_, _ = w.Write([]byte(err.Error()))
	}
	logger.Alert(err)
}

func (ln *Listener) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if scc.Stopped() {
		ln.HTTPErrorHandler(w, r, errors.New("server is stopped"))
		return
	}
	if ln.route != "" && r.URL.Path != ln.route {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("404 page not found"))
		return
	}

	var header = map[string][]string{"Sec-WebSocket-Protocol": {r.Header.Get("Sec-WebSocket-Protocol")}}

	conn, err := Options.Upgrader.Upgrade(w, r, header)
	if err != nil {
		ln.HTTPErrorHandler(w, r, err)
		return
	}
	// 使用非阻塞的方式发送连接到通道
	select {
	case ln.connChan <- conn:
	default:
		// 通道已满，关闭连接
		ln.HTTPErrorHandler(w, r, errors.New("connection channel full"))
		conn.Close()
	}

}
