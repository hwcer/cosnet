# cosnet

cosnet 是一个基于 Go 语言开发的高性能网络通信库，支持 TCP、UDP 和 WebSocket (WSS) 连接管理和消息处理。它提供了一套完整的网络通信框架，包括连接管理、消息编解码、路由分发和事件处理等功能。

## 功能特性

- **高性能网络通信**：基于 Go 语言的 goroutine 和通道实现异步处理，提高并发性能
- **灵活的消息格式**：支持路径型和代码型消息，适应不同场景
- **自动心跳机制**：保持连接活跃，检测死连接
- **事件驱动**：通过事件系统处理连接、断开等事件，提高代码的可扩展性
- **消息池管理**：复用消息对象，减少内存分配和 GC 压力
- **可扩展的路由**：基于注册中心的路由机制，支持灵活的消息分发
- **完善的错误处理**：提高系统的稳定性

## 项目结构

```
cosnet/
├── listener/          # 监听器接口定义
│   └── define.go
├── message/           # 消息处理相关
│   ├── head.go        # 消息头处理
│   ├── magic.go       # 魔术数字定义
│   ├── message.go     # 消息结构实现
│   ├── message_test.go # 消息测试
│   ├── options.go     # 消息选项
│   ├── pool.go        # 消息池
│   └── transform.go   # 消息转换
├── tcp/               # TCP 实现
│   ├── conn.go        # TCP 连接
│   └── listener.go    # TCP 监听器
├── udp/               # UDP 实现
│   ├── conn.go        # UDP 连接
│   └── listener.go    # UDP 监听器
├── wss/               # WebSocket 实现
│   ├── conn.go        # WebSocket 连接
│   └── listener.go    # WebSocket 监听器
├── context.go         # 上下文
├── cosnet.go          # 核心功能
├── events.go          # 事件系统
├── go.mod             # 依赖管理
├── go.sum             # 依赖校验
├── handler.go         # 消息处理器
├── heartbeat.go       # 心跳机制
├── options.go         # 选项配置
├── server.go          # 服务器实现
└── socket.go          # Socket 实现
```

## 安装

```bash
go get github.com/hwcer/cosnet
```

## 快速开始

### 服务器端示例

```go
package main

import (
    "fmt"
    "github.com/hwcer/cosnet"
)

func main() {
    // 注册消息处理器
    service := cosnet.Service()
    service.Register(&Handler{})
    
    // 监听 TCP 连接
    _, err := cosnet.Listen("tcp://0.0.0.0:8080")
    if err != nil {
        fmt.Println("监听失败:", err)
        return
    }
    
    // 监听 UDP 连接
    _, err = cosnet.Listen("udp://0.0.0.0:8081")
    if err != nil {
        fmt.Println("UDP 监听失败:", err)
        return
    }
    
    // 监听 WebSocket 连接
    _, err = cosnet.Listen("ws://0.0.0.0:8082")
    if err != nil {
        fmt.Println("WebSocket 监听失败:", err)
        return
    }
    
    // 启动服务器（实际项目中，这里应该启动 cosgo 框架）
    select {}
}

// Handler 消息处理器
type Handler struct{}

// Echo 处理 echo 消息
func (h *Handler) Echo(c *cosnet.Context) any {
    var req string
    if err := c.Bind(&req); err != nil {
        return err
    }
    return req
}
```

### 客户端示例

```go
package main

import (
    "fmt"
    "github.com/hwcer/cosnet"
)

func main() {
    // 连接服务器
    socket, err := cosnet.Connect("tcp://127.0.0.1:8080")
    if err != nil {
        fmt.Println("连接失败:", err)
        return
    }
    
    // 发送消息
    socket.Send(1, "echo", "hello world")
    
    // 等待响应
    select {}
}
```

## 核心 API

### 服务器相关

- **`Listen(address string, tlsConfig ...*tls.Config) (listener listener.Listener, err error)`**：
  监听指定地址的连接，支持 TCP、UDP 和 WebSocket 协议
  - `address`：监听地址，格式为 `scheme://host:port`，其中 scheme 可以是 `tcp`、`udp`、`ws` 或 `wss`
  - `tlsConfig`：TLS 配置，仅用于 WebSocket (wss) 协议

- **`Connect(address string) (socket *Socket, err error)`**：
  连接到指定地址的服务器

- **`New(conn listener.Conn) (socket *Socket, err error)`**：
  创建新的 Socket 并添加到管理器

### Socket 相关

- **`(sock *Socket) Send(index int32, path string, data any)`**：
  发送消息到客户端

- **`(sock *Socket) Close(delay ...int32)`**：
  关闭连接

- **`(sock *Socket) Authentication(v *session.Data, reconnect ...bool)`**：
  认证连接并绑定用户数据

### 消息相关

- **`Service(name ...string) *registry.Service`**：
  创建或获取服务

- **`Register(i interface{}, prefix ...string) error`**：
  注册处理器

## 事件系统

cosnet 提供了以下事件类型：

- **`EventTypeError`**：系统级别错误事件
- **`EventTypeMessage`**：所有未注册的消息事件
- **`EventTypeConnected`**：连接成功事件
- **`EventTypeReconnected`**：断线重连事件
- **`EventTypeAuthentication`**：身份认证事件，参数：是否重连
- **`EventTypeDisconnect`**：断开连接事件
- **`EventTypeReplaced`**：被顶号事件

### 注册事件处理器

```go
cosnet.On(cosnet.EventTypeConnected, func(socket *cosnet.Socket, data any) {
    fmt.Println("新连接:", socket.RemoteAddr())
})

cosnet.On(cosnet.EventTypeDisconnect, func(socket *cosnet.Socket, data any) {
    fmt.Println("断开连接:", socket.RemoteAddr())
})
```

## 配置选项

cosnet 提供了以下配置选项：

```go
var Options = struct {
    Heartbeat           int32 // 服务器心跳间隔，单位秒
    WriteChanSize       int32 // 写通道缓存大小
    ConnectMaxSize      int32 // 最大连接人数
    SocketConnectTime   int32 // 没有动作被判断为掉线的时间，单位秒
    SocketReplacedTime  int32 // 顶号延时关闭时间，单位秒
    ClientReconnectMax  int32 // 断线重连最大尝试次数
    ClientReconnectTime int32 // 断线重连每次等待时间，单位毫秒
}{
    Heartbeat:          2,           // 心跳间隔 2 秒
    WriteChanSize:      100,         // 写通道缓存 100 条消息
    ConnectMaxSize:     100000,      // 最大连接数 10 万
    SocketConnectTime:  30,          // 30 秒无动作判断为掉线
    SocketReplacedTime: 5,           // 顶号后 5 秒关闭旧连接
    ClientReconnectMax:  1000,       // 最大重连尝试 1000 次
    ClientReconnectTime: 5000,       // 每次重连等待 5 秒
}
```

## 心跳机制

cosnet 内置了心跳机制，用于保持连接活跃和检测死连接：

1. 服务器会定期（默认 2 秒）检查所有连接的心跳状态
2. 当连接超过一定时间（默认 30 秒）没有活动时，会被视为死连接并关闭
3. 任何消息的发送或接收都会重置心跳计数

## 性能优化

1. **使用消息池**：减少内存分配和 GC 压力
2. **异步处理**：使用 goroutine 处理读写操作，提高并发性能
3. **连接管理**：定期清理死连接，避免资源泄露
4. **序列化优化**：使用高效的序列化方式，如 JSON、Protocol Buffers 等

## 注意事项

1. **错误处理**：使用事件系统处理错误，不要在消息处理器中直接 panic
2. **资源管理**：及时关闭不再使用的连接，避免资源泄露
3. **消息大小**：注意控制消息大小，避免过大的消息影响性能
4. **并发安全**：在多 goroutine 环境下，注意并发安全问题

## 依赖项

- **github.com/hwcer/cosgo**：提供基础工具和框架
- **github.com/hwcer/logger**：日志库
- **golang.org/x/sync**：提供并发工具

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request 来改进这个项目。

## 联系方式

如果您有任何问题或建议，请通过 GitHub Issues 与我们联系。