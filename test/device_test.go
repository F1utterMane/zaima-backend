// Package test - 设备监控模块 (device handler) 的集成测试。
//
// 测试项:
//   - 数据上报: 正常首次上报、同日覆盖更新 (upsert)、缺少日期
//   - 日报: 正常获取、无数据时的兜底提示
//   - 月报: 存在报告、不存在报告
package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"zaima-backend/internal/model"
	"zaima-backend/internal/pkg/database"
)

// TestDeviceUpload_Success 验证正常的设备数据上报。
func TestDeviceUpload_Success(t *testing.T) {
	r := SetupTestRouter()
	uid := SeedElderUser(database.DB)
	token := GetTestToken(uid, 1)

	body, _ := json.Marshal(map[string]interface{}{
		"record_date":       "2026-03-05",
		"steps":             3000,
		"battery_level":     80,
		"screen_unlocks":    10,
		"screen_usage_mins": 60,
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/device/upload", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 验证 DB
	var log model.DeviceStatusLog
	database.DB.Where("user_id = ? AND record_date = ?", uid, "2026-03-05").First(&log)
	assert.Equal(t, 3000, log.Steps)
	assert.Equal(t, 80, log.BatteryLevel)
}

// TestDeviceUpload_Upsert 验证同一天重复上报会覆盖更新。
func TestDeviceUpload_Upsert(t *testing.T) {
	r := SetupTestRouter()
	uid := SeedElderUser(database.DB)
	token := GetTestToken(uid, 1)

	// 第一次上报
	body1, _ := json.Marshal(map[string]interface{}{
		"record_date": "2026-03-04",
		"steps":       1000,
	})
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/api/v1/device/upload", bytes.NewBuffer(body1))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w1, req1)

	// 第二次上报 (同日)
	body2, _ := json.Marshal(map[string]interface{}{
		"record_date": "2026-03-04",
		"steps":       5000,
	})
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/v1/device/upload", bytes.NewBuffer(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w2, req2)

	// 验证只有一条记录且步数已更新
	var logs []model.DeviceStatusLog
	database.DB.Where("user_id = ? AND record_date = ?", uid, "2026-03-04").Find(&logs)
	assert.Len(t, logs, 1, "同日应只有一条记录")
	assert.Equal(t, 5000, logs[0].Steps, "步数应被覆盖更新")
}

// TestDeviceUpload_MissingDate 验证缺少日期字段被拒。
func TestDeviceUpload_MissingDate(t *testing.T) {
	r := SetupTestRouter()
	uid := SeedElderUser(database.DB)
	token := GetTestToken(uid, 1)

	body := `{"steps":1000}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/device/upload", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestDailyInsight_NoData 验证无监控数据时的兜底响应。
func TestDailyInsight_NoData(t *testing.T) {
	r := SetupTestRouter()
	elderID := SeedElderUser(database.DB)
	youthID := SeedYouthUser(database.DB)
	token := GetTestToken(youthID, 2)

	// 建立绑定关系
	database.DB.Create(&model.UserRelation{
		ElderID: elderID, YouthID: youthID, InitiatorID: elderID, Status: 1,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/device/daily-insight?elder_id=%d", elderID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "no_data")
}

// TestDailyInsight_WithData 验证有数据时返回建议。
func TestDailyInsight_WithData(t *testing.T) {
	r := SetupTestRouter()
	elderID := SeedElderUser(database.DB)
	youthID := SeedYouthUser(database.DB)
	token := GetTestToken(youthID, 2)

	// 建立绑定关系
	database.DB.Create(&model.UserRelation{
		ElderID: elderID, YouthID: youthID, InitiatorID: elderID, Status: 1,
	})

	// 植入测试数据
	database.DB.Create(&model.DeviceStatusLog{
		UserID:          elderID,
		RecordDate:      "2026-03-05",
		Steps:           200,
		BatteryLevel:    50,
		ScreenUsageMins: 30,
	})

	url := fmt.Sprintf("/api/v1/device/daily-insight?elder_id=%d", elderID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "suggestion", "有数据时应返回 AI 建议")
}

// TestMonthlyReport_NotReady 验证月报不存在时返回提示。
func TestMonthlyReport_NotReady(t *testing.T) {
	r := SetupTestRouter()
	elderID := SeedElderUser(database.DB)
	youthID := SeedYouthUser(database.DB)
	token := GetTestToken(youthID, 2)

	// 建立绑定关系
	database.DB.Create(&model.UserRelation{
		ElderID: elderID, YouthID: youthID, InitiatorID: elderID, Status: 1,
	})

	url := fmt.Sprintf("/api/v1/device/monthly-report?elder_id=%d&month=2099-01", elderID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "not_ready")
}
