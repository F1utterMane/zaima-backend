// Package test - WebSocket Hub 的单元测试。
//
// 测试项:
//   - Hub 注册/注销连接
//   - IsOnline 判断
//   - SendToUser 消息投递
//   - 重复注册同一用户 (旧连接被关闭)
package test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"zaima-backend/internal/ws"
)

// TestHub_RegisterAndUnregister 验证用户注册上线和注销下线。
func TestHub_RegisterAndUnregister(t *testing.T) {
	hub := ws.NewHub()
	go hub.Run()
	defer time.Sleep(50 * time.Millisecond)

	client := &ws.Client{
		UserID: 100,
		Conn:   nil, // 单元测试不需要真实连接
		Send:   make(chan []byte, 16),
	}

	// 注册
	hub.Register <- client
	time.Sleep(50 * time.Millisecond)
	assert.True(t, hub.IsOnline(100), "注册后应在线")

	// 注销
	hub.Unregister <- client
	time.Sleep(50 * time.Millisecond)
	assert.False(t, hub.IsOnline(100), "注销后应离线")
}

// TestHub_IsOnline 验证在线状态判断。
func TestHub_IsOnline(t *testing.T) {
	hub := ws.NewHub()
	go hub.Run()

	assert.False(t, hub.IsOnline(999), "未注册用户应显示离线")

	client := &ws.Client{
		UserID: 999,
		Conn:   nil,
		Send:   make(chan []byte, 16),
	}
	hub.Register <- client
	time.Sleep(50 * time.Millisecond)

	assert.True(t, hub.IsOnline(999), "注册后应显示在线")
}

// TestHub_SendToUser_Online 验证向在线用户推送消息。
func TestHub_SendToUser_Online(t *testing.T) {
	hub := ws.NewHub()
	go hub.Run()

	client := &ws.Client{
		UserID: 200,
		Conn:   nil,
		Send:   make(chan []byte, 16),
	}
	hub.Register <- client
	time.Sleep(50 * time.Millisecond)

	msg := ws.WSMessage{
		Type:    "system",
		Content: "测试推送",
	}
	hub.SendToUser(200, msg)

	// 从 Send channel 读取消息
	select {
	case received := <-client.Send:
		assert.Contains(t, string(received), "测试推送", "应收到推送的消息内容")
	case <-time.After(1 * time.Second):
		t.Fatal("超时未收到消息")
	}
}

// TestHub_SendToUser_Offline 验证向离线用户推送不会 panic。
func TestHub_SendToUser_Offline(t *testing.T) {
	hub := ws.NewHub()
	go hub.Run()
	time.Sleep(50 * time.Millisecond)

	// 向不存在的用户发送消息，不应 panic
	msg := ws.WSMessage{
		Type:    "system",
		Content: "发给不存在的人",
	}
	assert.NotPanics(t, func() {
		hub.SendToUser(99999, msg)
	}, "向离线用户发送消息不应 panic")
}

// TestHub_DuplicateRegister 验证同一用户重复注册时旧连接被关闭。
func TestHub_DuplicateRegister(t *testing.T) {
	hub := ws.NewHub()
	go hub.Run()

	oldClient := &ws.Client{
		UserID: 300,
		Conn:   nil,
		Send:   make(chan []byte, 16),
	}
	newClient := &ws.Client{
		UserID: 300,
		Conn:   nil,
		Send:   make(chan []byte, 16),
	}

	// 注册旧连接
	hub.Register <- oldClient
	time.Sleep(50 * time.Millisecond)

	// 注册新连接 (同一 UserID)
	hub.Register <- newClient
	time.Sleep(50 * time.Millisecond)

	// 旧连接的 Send channel 应被关闭
	select {
	case _, ok := <-oldClient.Send:
		assert.False(t, ok, "旧连接的 Send channel 应被关闭")
	case <-time.After(1 * time.Second):
		t.Fatal("旧连接的 Send channel 未被关闭")
	}

	// 新连接仍然在线
	assert.True(t, hub.IsOnline(300), "新连接应在线")
}
