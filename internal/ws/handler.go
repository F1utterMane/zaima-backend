// Package ws - WebSocket 升级处理与连接入口。
package ws

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"zaima-backend/internal/config"
	"zaima-backend/internal/pkg/utils"
)

// upgrader WebSocket 升级器配置。
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 开发阶段允许所有来源
	CheckOrigin: func(r *http.Request) bool { return true },
}

// HandleWebSocket WebSocket 连接入口。
// GET /api/v1/ws?token=JWT_TOKEN
//
// 流程: 校验 Token -> 升级 HTTP 为 WebSocket -> 注册到 Hub -> 启动读写协程。
func HandleWebSocket(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从 query 参数获取 JWT Token
		token := c.Query("token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少认证 Token"})
			return
		}

		// 2. 验证 Token
		claims, err := utils.ParseToken(token, config.AppConfig.JWT.Secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token 无效或已过期"})
			return
		}

		// 3. 升级为 WebSocket 连接
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("[ws] 升级失败: %v", err)
			return
		}

		// 4. 创建 Client 并注册到 Hub
		client := &Client{
			UserID: claims.UserID,
			Conn:   conn,
			Send:   make(chan []byte, 256),
		}

		hub.Register <- client

		// 5. 启动读写协程
		go client.WritePump()
		go client.ReadPump(hub)
	}
}
