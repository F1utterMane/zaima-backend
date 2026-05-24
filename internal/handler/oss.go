// Package handler - OSS 文件上传授权相关接口。
package handler

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"zaima-backend/internal/pkg/response"
)

// ==================== 响应体定义 ====================

// OSSTokenResp OSS 直传授权响应。
type OSSTokenResp struct {
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	SecurityToken   string `json:"security_token,omitempty"`
	BucketName      string `json:"bucket_name"`
	Endpoint        string `json:"endpoint"`
	Expiration      string `json:"expiration"` // ISO 8601 格式时间
	Dir             string `json:"dir"`        // 推荐上传目录
	Signature       string `json:"signature"`  // 服务端签名
	Policy          string `json:"policy"`    // Base64 编码的 Policy
}

// OSSPolicy OSS 上传 Policy 结构体。
type OSSPolicy struct {
	Expiration string        `json:"expiration"`
	Conditions []interface{} `json:"conditions"`
}

// ==================== Handler ====================

// GetOSSToken 获取 OSS 上传授权令牌 (STS 直传模式)。
// GET /api/v1/oss/token
//
// 返回：
//   - 开发环境：返回简化的令牌信息（使用 mock 值）
//   - 生产环境：需集成阿里云 STS 服务获取临时凭证
//
// 安全性：
//   - 通过 Policy 限制上传目录、文件类型、大小等
//   - 令牌有效期为 1 小时
func GetOSSToken(c *gin.Context) {
	userID := c.GetUint64("user_id")

	// 令牌有效期（1小时）
	expiration := time.Now().Add(1 * time.Hour)
	expirationStr := expiration.UTC().Format("2006-01-02T15:04:05Z")

	// 推荐上传目录（按用户ID分类）
	dir := fmt.Sprintf("uploads/users/%d/", userID)

	// 配置（生产环境应从环境变量读取）
	accessKeyID := getEnvOrDefault("OSS_ACCESS_KEY_ID", "test-access-key-id")
	accessKeySecret := getEnvOrDefault("OSS_ACCESS_KEY_SECRET", "test-access-key-secret")
	bucketName := getEnvOrDefault("OSS_BUCKET_NAME", "zaima")
	endpoint := getEnvOrDefault("OSS_ENDPOINT", "https://zaima.oss.aliyuncs.com")

	// 构建 Policy (限制上传目录、文件类型、大小)
	conditions := []interface{}{
		[]interface{}{"content-length-range", 0, 10485760}, // 10MB 限制
		[]interface{}{"starts-with", "$key", dir},          // 限制上传目录
		[]interface{}{"in", "$content-type", []string{
			"image/jpeg", "image/png", "image/gif", "image/webp", // 图片
			"audio/mpeg", "audio/wav", "audio/ogg",               // 音频
		}},
	}

	policy := OSSPolicy{
		Expiration: expirationStr,
		Conditions: conditions,
	}

	policyJSON, _ := json.Marshal(policy)
	policyBase64 := base64.StdEncoding.EncodeToString(policyJSON)

	// 生成签名
	h := hmac.New(sha1.New, []byte(accessKeySecret))
	h.Write([]byte(policyBase64))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	response.OK(c, OSSTokenResp{
		AccessKeyID:     accessKeyID,
		AccessKeySecret: accessKeySecret,
		BucketName:      bucketName,
		Endpoint:        endpoint,
		Expiration:      expirationStr,
		Dir:             dir,
		Signature:       signature,
		Policy:          policyBase64,
	})
}

// ValidateOSSURL 校验 OSS URL 是否合法（防止 SSRF）。
// 在 user.go 和 square.go 中被使用。
func ValidateOSSURLFormat(url string) bool {
	if url == "" {
		return false
	}

	// 必须是 HTTPS 协议
	if len(url) < 8 || url[:8] != "https://" {
		return false
	}

	// 白名单: 允许通过的域名模式
	allowedPatterns := []string{
		"aliyuncs.com/",
		"myqcloud.com/",
		"amazonaws.com/",
		"cdn.",
		"oss.",
	}
	for _, pattern := range allowedPatterns {
		if containsString(url, pattern) {
			return true
		}
	}

	// 开发阶段兜底: 允许 example.com 用于测试
	if containsString(url, "example.com") {
		return true
	}
	return false
}

// ==================== 工具函数 ====================

// getEnvOrDefault 获取环境变量或返回默认值。
func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// containsString 字符串包含检查。
func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) && (s == substr || contains(s, substr))
}
