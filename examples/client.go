package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hwcer/cosnet"
	"github.com/hwcer/cosnet/message"
)

func main() {
	// 启动cosnet
	if err := cosnet.Start(); err != nil {
		log.Fatalf("启动cosnet失败: %v", err)
	}

	// 注册消息处理函数
	cosnet.On(cosnet.EventTypeMessage, func(s *cosnet.Socket, data any) {
		if msg, ok := data.(message.Message); ok {
			path, _, err := msg.Path()
			if err != nil {
				fmt.Printf("解析消息路径失败: %v\n", err)
				return
			}

			// 处理不同类型的消息
			switch path {
			case "pubsub/message":
				var msgData struct {
					Topic   string      `json:"topic"`
					Message interface{} `json:"message"`
				}
				if err := msg.Unmarshal(&msgData); err != nil {
					fmt.Printf("解析消息数据失败: %v\n", err)
					return
				}
				fmt.Printf("收到主题[%s]的消息: %v\n", msgData.Topic, msgData.Message)
			default:
				fmt.Printf("收到未知消息: %v\n", path)
			}
		}
	})

	// 连接到TCP服务器
	socket, err := cosnet.Connect("tcp://127.0.0.1:8888")
	if err != nil {
		log.Fatalf("连接服务器失败: %v", err)
	}
	defer socket.Close()

	fmt.Println("已连接到服务器")

	// 发送订阅请求
	socket.Send(0, 0, "pubsub/subscribe", map[string]interface{}{
		"topics": []string{"test"},
	})
	fmt.Println("已发送订阅请求，订阅主题: test")

	// 发送发布请求
	socket.Send(0, 0, "pubsub/publish", map[string]interface{}{
		"topic":   "test",
		"message": "Hello from client",
	})
	fmt.Println("已发送发布请求，发布消息到主题: test")

	// 等待用户输入
	fmt.Println("按Enter键退出")
	bufio.NewReader(os.Stdin).ReadString('\n')

	// 发送取消订阅请求
	socket.Send(0, 0, "pubsub/unsubscribe", map[string]interface{}{
		"topics": []string{"test"},
	})
	fmt.Println("已发送取消订阅请求")

	time.Sleep(1 * time.Second)
	fmt.Println("客户端退出")
}
