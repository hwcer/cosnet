# cosnet

> [!WARNING]
> **碳基生物警告（Carbon-based Lifeform Warning）**
> 本文档由 AI 协作生成/重写，内容基于当前代码状态。API 描述、参数签名、默认值等请以源码为准；代码示例已尽量校准，但仍建议人工复核后再用于生产。

cosnet 是一个基于 Go 的高性能网络通信库，统一封装 TCP / UDP / WebSocket 三种传输，提供连接管理、消息编解码、路由分发和事件系统。

## 特性

- **三种传输统一接口**：TCP / UDP / WebSocket(WSS) 通过同一套 `Socket` API 使用。
- **双模式消息协议**：`path` 模式（字符串路由）与 `code` 模式（数字协议号），同一进程可混用，由消息魔数标记。
- **多序列化支持**：内置 JSON 和 Protobuf 绑定，通过魔数选择。
- **自动压缩**：消息体超过阈值自动 gzip，对端透明解压。
- **消息池**：`message.Message` 走 `sync.Pool`，减少 GC 压力。
- **异步写通道**：每个 Socket 独立写协程 + 缓冲 channel，Send 非阻塞返回。
- **心跳 + 掉线检测**：内置定时器，超时自动 disconnect。
- **客户端断线重连**：指数退避，可配置上限。
- **事件总线**：连接、断开、顶号、心跳、错误等钩子。

## 安装

```bash
go get github.com/hwcer/cosnet
```

## 快速开始

### 服务端

```go
package main

import (
    "github.com/hwcer/cosgo"
    "github.com/hwcer/cosnet"
)

type Handler struct{}

// Echo 处理 path = "/Handler/Echo" 的消息
func (h *Handler) Echo(c *cosnet.Context) any {
    var req string
    if err := c.Bind(&req); err != nil {
        return err
    }
    return req
}

func main() {
    // 注册处理器（按方法名自动路由）
    _ = cosnet.Register(&Handler{})

    // 多种协议可同时监听
    if _, err := cosnet.Listen("tcp://0.0.0.0:8080"); err != nil {
        panic(err)
    }
    if _, err := cosnet.Listen("ws://0.0.0.0:8082"); err != nil {
        panic(err)
    }

    // 通过 cosgo 框架启动（它会回调 cosnet.Start）
    cosgo.Start()
}
```

### 客户端

```go
package main

import (
    "github.com/hwcer/cosnet"
    "github.com/hwcer/cosnet/message"
)

func main() {
    sock, err := cosnet.Connect("tcp://127.0.0.1:8080")
    if err != nil {
        panic(err)
    }
    // flag=0 普通包, index=1 消息序号, path="/Handler/Echo"
    _ = sock.Send(0, 1, "/Handler/Echo", "hello")

    // 监听服务端响应
    cosnet.On(cosnet.EventTypeMessage, func(s *cosnet.Socket, data any) {
        msg := data.(message.Message)
        var reply string
        _ = msg.Unmarshal(&reply)
        println("recv:", reply)
    })

    select {}
}
```

## 核心概念

### 消息格式

```
+--------+--------+------------+------------+-------------------+-------------+
| magic  |  flag  |    size    |   index    | path(len-prefix)  |    body     |
| 1 byte | 1 byte |  4 bytes   |  4 bytes   |  or code(4 bytes) |   N bytes   |
+--------+--------+------------+------------+-------------------+-------------+
\---------------- 10 byte head ------------/
```

- `magic` 决定工作模式（path/code）、序列化方式和字节序，参见魔数表。
- `flag` 是位标志，见下。
- `size` 是 body（含 path/code 4 字节）总长，用于分帧。
- `index` 是客户端自定义的消息序号，服务端原值返回便于对应。

### 魔数表（`message/magic.go`）

| Magic | 模式 | 序列化 | 字节序 | 备注 |
|-------|------|--------|--------|------|
| `0xf0` `MagicNumberPathJson`  | path | JSON     | BigEndian | 默认 |
| `0xf1` `MagicNumberCodeJson`  | code | JSON     | BigEndian |      |
| `0xf2` `MagicNumberCodeProto` | code | Protobuf | BigEndian |      |

默认魔数：`message.Options.Magic = MagicNumberPathJson`。可通过 `sock.Magic(0xf1)` 或 `SendWithMagic(...)` 单次覆盖。

### path 模式 vs code 模式

- **path 模式**：head 之后是 `uint32 pathLen` + `pathLen` 字节的 UTF-8 路径字符串，例如 `/Handler/Echo`。路由直接按路径匹配。
- **code 模式**：head 之后是 4 字节数字协议号。需要通过 `message.Transform` 提供 `code ↔ path` 的双向映射：

```go
type myTransform struct{}
func (myTransform) Path(code int32) (string, error) { /* code → "/..." */ }
func (myTransform) Code(path string) (int32, error) { /* "/..." → code */ }

// 启动期一次性注入
message.Transform = myTransform{}
```

未注入 Transform 时，code 模式下任何编解码都会返回 `ErrMsgHeadNotSetTransform`。

### Flag 位标志（`message/flag.go`）

| Flag | 含义 |
|------|------|
| `FlagConfirm`    | 此包是对请求的确认/响应 |
| `FlagNoreply`    | 服务端处理后不需要回包 |
| `FlagHeartbeat`  | 心跳包（与普通消息一样会重置 heartbeat 计数）|
| `FlagBroadcast`  | 广播包 |
| `FlagCompressed` | body 已 gzip（由库自动管理，一般无需手动设置）|
| `FlagEncrypted`  | 已加密（库本身不实现加密，需业务层处理）|
| `FlagFragmented` | 分片包 |

### Socket 发送

`Send` 签名：

```go
func (sock *Socket) Send(flag message.Flag, index int32, path any, data any, safe ...bool) error
```

- `path any` 接受 `string`（走 path 模式）或 `int/int32/int64/uint/uint32/uint64`（走 code 模式）。实际路径模式取决于当前 `magic` 对应的 `MagicType`——如果魔数是 code 模式但传了 string，会走 `Transform.Code(path)` 转换；反之同理。
- `data any`：`[]byte`/`*[]byte` 直接透传，其它类型走当前魔数的 Binder 序列化。
- `safe`：默认 `true`，写通道满时阻塞等待；`false` 时满则丢弃并返回 `channel full` 错误。

> Send 在 Marshal 或 Write 失败时会自动把消息对象归还到池中。若你用 `Async(m)` 或 `Write(m)` 直接发送自己构造的消息，需要自行管理 `m` 的生命周期（通常由发送协程 defer `message.Release`，失败时也要释放）。

### Handler 与 Context

服务端通过 `Register(obj)` 将对象上的方法自动注册到路由表，方法签名须为：

```go
func (h *Handler) Method(c *cosnet.Context) any
```

- `*cosnet.Context` 内嵌 `*Socket`，并带有当前 `Message`。
- `c.Bind(&req)` 将 body 反序列化到结构体。
- 返回值 `any` 会被自动序列化并作为 `FlagConfirm` 包回传；返回 `error` 时当作错误回传。
- 若请求带 `FlagNoreply` 或本身是 `FlagConfirm`，则不回包。

### 事件系统

```go
cosnet.On(cosnet.EventTypeConnected, func(s *cosnet.Socket, _ any) {
    println("connected:", s.RemoteAddr().String())
})
```

| 事件 | 触发时机 | attach 参数 |
|------|----------|-------------|
| `EventTypeError`          | 框架级错误 | error/字符串 |
| `EventTypeHeartbeat`      | 每次心跳 tick | 心跳增量 `int32` |
| `EventTypeMessage`        | 收到**未注册路径**的消息 | `message.Message` |
| `EventTypeConnected`      | 连接建立 | nil |
| `EventTypeReconnected`    | 客户端断线重连成功 | nil |
| `EventTypeDisconnect`     | 连接断开 | nil |
| `EventTypeAuthentication` | 调用 `Authentication()` | `bool` 是否重连 |
| `EventTypeReplaced`       | 被顶号 | 新登录者 IP `string` |

事件回调建议在**启动前**注册；运行期修改 `emitter` 无锁保护。

### Socket 生命周期

```
  None ──connect──► Connected ──Close()──► Closing ──heartbeat超时──► Disconnect
                        │                                                  │
                        └─────────────readMsg/writeMsg error───────────────┘
                                                                           ▼
                      客户端：Reconnecting ──成功──► Connected          Disconnected
                                           └──失败──► Released                │
                                                                           ▼
                                                                        Released
```

## 配置

```go
cosnet.Options = cosnet.Config{
    Heartbeat:               10,     // 心跳 tick 间隔（秒），0 表示不启动 daemon
    WriteChanSize:           100,    // 每个 Socket 写通道缓冲
    ConnectMaxSize:          100000, // 最大并发连接，0 不限
    SocketConnectTime:       30,     // 无活动多少秒判定掉线
    SocketReplacedTime:      5,      // 被顶号延时关闭旧连接（秒）
    ClientReconnectMax:      10,     // 客户端最大重连次数，0 无限
    ClientReconnectTime:     1000,   // 重连基础等待（毫秒），实际为指数退避
    ClientReconnectMaxDelay: 30000,  // 重连等待上限（毫秒）
}
```

消息层配置（`message.Options`）：

```go
message.Options.Magic            = message.MagicNumberPathJson // 默认魔数
message.Options.Pool             = true                        // 启用消息池
message.Options.Capacity         = 1024                        // 单条消息初始/回收容量
message.Options.MaxDataSize      = 1024 * 1024                 // 单包最大 size，超过 Parse 报错
message.Options.AutoCompressSize = 1024 * 100                  // 超过此字节自动 gzip，0 关闭
message.Options.S2CConfirm       = ""                          // 默认确认包路径；空则原路返回
```

## 协议与消息

### 自定义 Transform（code 模式必需）

```go
// 示例：code 高 16 位服务号 + 低 16 位方法号
type codec struct{}

func (codec) Code(path string) (int32, error) {
    // /svc/method → int32
}
func (codec) Path(code int32) (string, error) {
    // int32 → /svc/method
}

func init() {
    message.Transform = codec{}
}
```

### 自定义 Handler 序列化

```go
h := cosnet.Default.Handler() // 默认 Handler
h.SetSerialize(func(c *cosnet.Context, reply any) ([]byte, error) {
    return json.Marshal(reply) // 例：强制 JSON，无视当前 Binder
})
```

### 广播

```go
m := message.Require()
defer message.Release(m)
_ = m.Marshal(message.Options.Magic, message.FlagBroadcast, 0, "/notify", payload)

cosnet.Broadcast(m, func(s *cosnet.Socket) bool {
    return s.Data() != nil // 只发给已认证的连接
})
```

> `Broadcast` 会对同一份 `m` 并发调用 `Async`，message 由多个 goroutine 共享只读；构造好之后不要再修改它。

## 连接管理

- `sock.Close(delay...)` 把状态置为 `Closing`，在 `delay` 秒后由心跳协程真正断开。期间 `cwrite` 里已排队的消息会继续发完。
- `sock.Authentication(data, reconnect...)` 绑定 `session.Data`，触发 `EventTypeAuthentication`，重连场景额外触发 `EventTypeReconnected`。
- `sock.Replaced(newIP)` 处理顶号：清除 `data`，`SocketReplacedTime` 秒后关闭旧连接。
- `sock.KeepAlive()` 手动重置心跳计数（收到业务消息时会自动调用）。

## 注意事项

1. **资源回收**：用 `message.Require()` 拿到的消息，只要交给 `Send/Write/Async` 之一，由库负责释放；其它情况要自己 `defer message.Release(m)`。
2. **code 模式必须先注入 Transform**，否则任何 code 模式消息的编解码会直接报错。
3. **`MaxDataSize` 是 head 解析层的硬上限**，超过会返回 `ErrMsgDataSizeTooLong` 并切断连接——生产环境务必根据业务最大包大小配置，避免被畸形包拖垮。
4. **`EventTypeMessage` 仅在路径未注册时触发**。已注册的消息会走 Handler 链，不再派发该事件。
5. **事件回调不要阻塞**：它在触发消息的协程里同步执行，阻塞会卡住 readMsg。
6. **UDP 是伪连接**：listener 会为每个 remote addr 合成一个 Socket，心跳同样生效。

## 协议兼容性说明

当前魔数 `0xf2 MagicNumberCodeProto` 替换了早期版本的 `0xf9 MagicNumberPathBytes`（Bytes + LittleEndian）。若需要跨版本互通，请锁定通信两端的 cosnet 版本。

## 依赖

- `github.com/hwcer/cosgo`    — 基础框架、协程管理
- `github.com/hwcer/logger`   — 日志
- `golang.org/x/sync/syncmap` — 并发 map

## 许可证

MIT
