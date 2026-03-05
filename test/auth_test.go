// Package test - 认证模块 (auth handler) 的集成测试。
//
// 测试项:
//   - 发送验证码: 正常、手机号为空、手机号长度错误
//   - 登录: 正常新用户注册、验证码错误、角色非法、老用户登录
package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"zaima-backend/internal/pkg/database"
	"zaima-backend/internal/pkg/response"
)

// TestSendSMSCode_Success 验证正常发送验证码流程。
func TestSendSMSCode_Success(t *testing.T) {
	r := SetupTestRouter()

	body := `{"phone":"13800138000"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/sms-code", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp response.R
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 0, resp.Code, "业务码应为 0")
}

// TestSendSMSCode_EmptyPhone 验证空手机号被拒绝。
func TestSendSMSCode_EmptyPhone(t *testing.T) {
	r := SetupTestRouter()

	body := `{"phone":""}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/sms-code", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "空手机号应返回 400")
}

// TestSendSMSCode_BadPhoneLength 验证非11位手机号被拒绝。
func TestSendSMSCode_BadPhoneLength(t *testing.T) {
	r := SetupTestRouter()

	body := `{"phone":"123"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/sms-code", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "非11位应返回 400")
}

// TestLogin_NewUser 验证新用户注册+登录全流程。
func TestLogin_NewUser(t *testing.T) {
	r := SetupTestRouter()
	phone := "13811112222"

	// 1. 先发送验证码
	sendBody := `{"phone":"` + phone + `"}`
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/api/v1/auth/sms-code", bytes.NewBufferString(sendBody))
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w1, req1)
	require.Equal(t, http.StatusOK, w1.Code)

	// 从 Redis 取出验证码
	code, err := database.RDB.Get(context.Background(), "sms:code:"+phone).Result()
	require.NoError(t, err, "Redis 中应存在验证码")

	// 2. 使用验证码登录
	loginBody, _ := json.Marshal(map[string]interface{}{
		"phone": phone,
		"code":  code,
		"role":  2,
	})
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginBody))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	var resp map[string]interface{}
	_ = json.Unmarshal(w2.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.NotEmpty(t, data["token"], "应返回 JWT Token")
	assert.Equal(t, true, data["is_new"], "应标记为新用户")
}

// TestLogin_WrongCode 验证错误验证码被拒绝。
func TestLogin_WrongCode(t *testing.T) {
	r := SetupTestRouter()

	body, _ := json.Marshal(map[string]interface{}{
		"phone": "13800138000",
		"code":  "000000",
		"role":  1,
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp response.R
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 1001, resp.Code, "错误验证码业务码应为 1001")
}

// TestLogin_InvalidRole 验证非法角色值被拒绝。
func TestLogin_InvalidRole(t *testing.T) {
	r := SetupTestRouter()

	body, _ := json.Marshal(map[string]interface{}{
		"phone": "13800138000",
		"code":  "123456",
		"role":  3, // 非法角色
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "非法角色应返回 400")
}
