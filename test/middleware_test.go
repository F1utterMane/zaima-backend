// Package test - JWT 鉴权中间件与 CORS 中间件的单元测试。
//
// 测试项:
//   - 缺少 Authorization 头 -> 401
//   - 错误格式的 Authorization 头 -> 401
//   - 携带无效 Token -> 401
//   - 携带合法 Token -> 200 并注入上下文
//   - CORS 中间件 OPTIONS 预检 -> 204
//   - CORS 响应头验证
package test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"zaima-backend/internal/config"
	"zaima-backend/internal/middleware"
	"zaima-backend/internal/pkg/utils"
)

// setupAuthRouter 构建一个带 JWTAuth 中间件的测试路由。
// 使用共享的 TestSecret 常量确保 Token 签名一致。
func setupAuthRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	config.AppConfig = &config.Config{
		JWT: config.JWTConfig{
			Secret:      TestSecret,
			ExpireHours: 24,
		},
	}

	r := gin.New()
	r.GET("/protected", middleware.JWTAuth(), func(c *gin.Context) {
		userID := c.GetUint64("user_id")
		role := c.GetInt("role")
		c.JSON(200, gin.H{"user_id": userID, "role": role})
	})
	return r
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	r := setupAuthRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "缺少 Auth Header 应返回 401")
}

func TestJWTAuth_BadFormat(t *testing.T) {
	r := setupAuthRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Basic abc123") // 错误格式
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "非 Bearer 格式应返回 401")
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	r := setupAuthRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.string")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "无效 Token 应返回 401")
}

func TestJWTAuth_ValidToken(t *testing.T) {
	token, _ := utils.GenerateToken(99, "13800000001", 2, TestSecret, 24)

	r := setupAuthRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "合法 Token 应返回 200")
	assert.Contains(t, w.Body.String(), `"user_id":99`, "应注入正确的 user_id")
	assert.Contains(t, w.Body.String(), `"role":2`, "应注入正确的 role")
}

func TestCORS_OptionsRequest(t *testing.T) {
	r := gin.New()
	r.Use(middleware.CORS())
	r.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 204, w.Code, "OPTIONS 预检应返回 204")
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_NormalRequest(t *testing.T) {
	r := gin.New()
	r.Use(middleware.CORS())
	r.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code, "GET 请求应返回 200")
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}
