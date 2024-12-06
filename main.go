package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"testapi/feishu"

	"github.com/gin-gonic/gin"
)

func main() {
	// 创建上下文用于控制定时任务的生命周期
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // 确保在函数退出时取消上下文

	// 创建计时器
	ticker := time.NewTicker(30 * time.Second) // 为了测试，减少间隔时间
	defer ticker.Stop()                        // 确保在函数退出时停止 ticker

	// 使用 goroutine 执行定时任务
	go func() {
		for {
			select {
			case <-ctx.Done(): // 上下文被取消时退出
				fmt.Println("Task 1: Context cancelled, exiting...")
				return
			case <-ticker.C:
				fmt.Println("Task 1: Calling GetInstancekubeCodeList")
				feishu.GetInstancekubeCodeList()
			}
		}
	}()

	// 设置 Gin 路由
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// 启动 HTTP 服务器
	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		addr := fmt.Sprintf(":%s", port)
		fmt.Printf("Server started at http://127.0.0.1%s\n", addr)
		if err := r.Run(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 优雅地关闭服务器
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown

	// 取消上下文，停止定时任务
	cancel()
	fmt.Println("Shutting down server...")

	// 等待一段时间以确保定时任务有机会停止
	time.Sleep(5 * time.Second) // 可以根据需要调整等待时间
}
