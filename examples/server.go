package main

import (
	"fmt"
	"log"
	"time"

	"github.com/hwcer/cosnet"
	"github.com/hwcer/cosnet/pubsub"
)

func main() {
	// 初始化pubsub系统
	pubsub.Init()

	// 启动cosnet
	if err := cosnet.Start(); err != nil {
		log.Fatalf("启动cosnet失败: %v", err)
	}

	// 启动TCP服务器
	listener, err := cosnet.Listen("tcp://0.0.0.0:8888")
	if err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
	defer listener.Close()

	// 模拟定期发布消息
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			<-ticker.C
			// 发布消息到"test"主题
			pubsub.PubSub.Publish("test", map[string]interface{}{
				"message": "Hello from server",
				"time":    time.Now().Format(time.RFC3339),
			})
			fmt.Println("已发布消息到test主题")
		}
	}()

	fmt.Println("TCP服务器已启动，监听端口8888")
	fmt.Println("按Ctrl+C退出")

	// 等待退出
	select {}
}
