// Package main 是 "在吗" APP 后端服务的入口。
//
// 启动流程:
//  1. 加载配置文件 (configs/config.yaml)
//  2. 初始化 PostgreSQL + Redis 连接
//  3. 启动 WebSocket Hub (独立 goroutine)
//  4. 注册路由并启动 HTTP 服务
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"zaima-backend/internal/config"
	"zaima-backend/internal/pkg/database"
	"zaima-backend/internal/router"
	"zaima-backend/internal/ws"
)

func main() {
	// 命令行参数：指定配置文件路径
	configPath := flag.String("config", "configs/config.yaml", "配置文件路径")
	flag.Parse()

	// 1. 加载配置
	config.Load(*configPath)

	// 2. 设置 Gin 模式
	gin.SetMode(config.AppConfig.Server.Mode)

	// 3. 初始化数据库连接
	database.InitPostgres(&config.AppConfig.Database)
	database.InitRedis(&config.AppConfig.Redis)

	// 4. 启动 WebSocket Hub
	hub := ws.NewHub()
	go hub.Run()

	// 5. 初始化路由并启动 HTTP 服务
	r := router.SetupRouter(hub)

	addr := fmt.Sprintf(":%d", config.AppConfig.Server.Port)
	log.Printf("[main] 🚀 在吗后端服务启动: http://localhost%s", addr)
	log.Printf("[main] 📋 API 文档: http://localhost%s/health", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("[main] 服务启动失败: %v", err)
	}
}
