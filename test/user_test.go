// Package test - 用户模块 (user handler) 的集成测试。
//
// 测试项:
//   - 获取用户资料: 正常、无鉴权
//   - 更新用户资料: 正常、昵称超长
//   - 兴趣标签: 正常更新、超过3个
//   - 亲子绑定: 发起绑定、确认绑定、对方未注册
package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"zaima-backend/internal/model"
	"zaima-backend/internal/pkg/database"
)

// TestGetProfile_Unauthorized 验证无 Token 访问返回 401。
func TestGetProfile_Unauthorized(t *testing.T) {
	r := SetupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/user/profile", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestGetProfile_Success 验证正常获取用户资料。
func TestGetProfile_Success(t *testing.T) {
	r := SetupTestRouter()
	uid := SeedYouthUser(database.DB)
	token := GetTestToken(uid, 2)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/user/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"nickname"`, "应返回用户资料")
}

// TestUpdateProfile_Success 验证正常更新资料。
func TestUpdateProfile_Success(t *testing.T) {
	r := SetupTestRouter()
	uid := SeedElderUser(database.DB)
	token := GetTestToken(uid, 1)

	body := `{"nickname":"王奶奶","city":"北京"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/user/profile", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 验证数据库已更新
	var user model.User
	database.DB.First(&user, uid)
	assert.Contains(t, user.Nickname, "王奶奶")
	assert.Contains(t, user.City, "北京")
}

// TestUpdateProfile_NicknameTooLong 验证超长昵称被拒。
func TestUpdateProfile_NicknameTooLong(t *testing.T) {
	r := SetupTestRouter()
	uid := SeedElderUser(database.DB)
	token := GetTestToken(uid, 1)

	body := `{"nickname":"超级无敌长昵称要被拒绝的"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/user/profile", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "1003", "超长昵称应返回业务码 1003")
}

// TestUpdateInterests_Success 验证正常更新兴趣标签。
func TestUpdateInterests_Success(t *testing.T) {
	r := SetupTestRouter()
	uid := SeedElderUser(database.DB)
	token := GetTestToken(uid, 1)

	body := `{"tags":["棋牌","广场舞"]}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/user/interests", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 验证数据库
	var interests []model.UserInterest
	database.DB.Where("user_id = ? AND status = 1", uid).Find(&interests)
	assert.Len(t, interests, 2)

	// 再次更新时不删除历史记录，只将旧标签置为历史并插入新标签
	body = `{"tags":["太极拳"]}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/api/v1/user/interests", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var allRows int64
	database.DB.Model(&model.UserInterest{}).Where("user_id = ?", uid).Count(&allRows)
	assert.Equal(t, int64(3), allRows, "历史兴趣记录不应被删除")

	var active []model.UserInterest
	database.DB.Where("user_id = ? AND status = 1", uid).Find(&active)
	assert.Len(t, active, 1)
	assert.Equal(t, "太极拳", active[0].InterestTag)
}

// TestUpdateInterests_TooMany 验证超过3个兴趣标签被拒。
func TestUpdateInterests_TooMany(t *testing.T) {
	r := SetupTestRouter()
	uid := SeedElderUser(database.DB)
	token := GetTestToken(uid, 1)

	body := `{"tags":["棋牌","广场舞","散步","钓鱼"]}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/user/interests", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "1006")
}

// TestBindRequest_TargetNotRegistered 验证对方未注册时的流程。
func TestBindRequest_TargetNotRegistered(t *testing.T) {
	r := SetupTestRouter()
	uid := SeedElderUser(database.DB)
	token := GetTestToken(uid, 1)

	body := `{"target_phone":"19999999999"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/user/bind", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "not_registered")
}

// TestBindRequest_AndConfirm 验证完整的绑定+确认流程。
func TestBindRequest_AndConfirm(t *testing.T) {
	r := SetupTestRouter()
	elderID := SeedElderUser(database.DB)
	youthID := SeedYouthUser(database.DB)

	// 查出年轻人手机号
	var youth model.User
	database.DB.First(&youth, youthID)

	// 1. 老人发起绑定
	elderToken := GetTestToken(elderID, 1)
	bindBody, _ := json.Marshal(map[string]interface{}{
		"target_phone": youth.Phone,
		"remark":       "儿子",
	})
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/api/v1/user/bind", bytes.NewBuffer(bindBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+elderToken)
	r.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Contains(t, w1.Body.String(), "pending")

	// 提取 relation_id
	var resp1 map[string]interface{}
	_ = json.Unmarshal(w1.Body.Bytes(), &resp1)
	data1, ok := resp1["data"].(map[string]interface{})
	if !ok || data1 == nil {
		t.Fatal("响应中缺少 data 字段")
	}
	relationIDFloat, ok := data1["relation_id"].(float64)
	if !ok {
		t.Fatal("响应中缺少 relation_id")
	}
	relationID := uint64(relationIDFloat)

	// 2. 年轻人确认绑定
	youthToken := GetTestToken(youthID, 2)
	confirmBody, _ := json.Marshal(map[string]interface{}{
		"relation_id": relationID,
	})
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/v1/user/bind/confirm", bytes.NewBuffer(confirmBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+youthToken)
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "绑定成功")

	// 3. 验证数据库状态
	var relation model.UserRelation
	database.DB.First(&relation, relationID)
	assert.Equal(t, 1, relation.Status, "状态应更新为已绑定(1)")
}
