// Package test - JWT 工具函数的单元测试。
//
// 测试项:
//   - Token 生成与解析的完整正向流程
//   - 使用错误密钥解析应失败
//   - 过期 Token 解析应失败
//   - Claims 中自定义字段 (UserID, Phone, Role) 正确性
//   - 生成的验证码始终为6位数字
package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"zaima-backend/internal/pkg/utils"
)

// TestGenerateAndParseToken 验证 Token 的完整生命周期:
// 生成 -> 解析 -> Claims 字段校验。
func TestGenerateAndParseToken(t *testing.T) {
	secret := "unit-test-secret"
	userID := uint64(42)
	phone := "13800000001"
	role := 1

	token, err := utils.GenerateToken(userID, phone, role, secret, 24)
	require.NoError(t, err, "Token 生成不应报错")
	assert.NotEmpty(t, token, "Token 不应为空字符串")

	// 解析
	claims, err := utils.ParseToken(token, secret)
	require.NoError(t, err, "使用正确密钥解析 Token 不应报错")
	assert.Equal(t, userID, claims.UserID, "UserID 应与生成时一致")
	assert.Equal(t, phone, claims.Phone, "Phone 应与生成时一致")
	assert.Equal(t, role, claims.Role, "Role 应与生成时一致")
	assert.Equal(t, "zaima-backend", claims.Issuer, "Issuer 应为 zaima-backend")
}

// TestParseTokenWithWrongSecret 验证使用错误密钥无法解析 Token。
func TestParseTokenWithWrongSecret(t *testing.T) {
	token, _ := utils.GenerateToken(1, "13800000001", 1, "correct-secret", 24)

	_, err := utils.ParseToken(token, "wrong-secret")
	assert.Error(t, err, "使用错误密钥解析应报错")
}

// TestParseTokenExpired 验证过期 Token 解析失败。
func TestParseTokenExpired(t *testing.T) {
	// 生成有效期为 0 小时的 Token (立刻过期)
	token, _ := utils.GenerateToken(1, "13800000001", 1, "secret", 0)

	_, err := utils.ParseToken(token, "secret")
	assert.Error(t, err, "过期 Token 解析应报错")
}

// TestParseTokenInvalidString 验证乱码字符串解析失败。
func TestParseTokenInvalidString(t *testing.T) {
	_, err := utils.ParseToken("this.is.not.a.jwt", "secret")
	assert.Error(t, err, "非法字符串解析应报错")
}

// TestGenerateSMSCode 验证验证码格式。
func TestGenerateSMSCode(t *testing.T) {
	for i := 0; i < 100; i++ {
		code := utils.GenerateSMSCode()
		assert.Len(t, code, 6, "验证码必须为6位")
		// 确保全部是数字
		for _, ch := range code {
			assert.True(t, ch >= '0' && ch <= '9', "验证码每一位都应是数字")
		}
	}
}

// TestGenerateSMSCodeUniqueness 验证连续生成的验证码具有一定随机性。
func TestGenerateSMSCodeUniqueness(t *testing.T) {
	codes := map[string]bool{}
	for i := 0; i < 50; i++ {
		codes[utils.GenerateSMSCode()] = true
	}
	// 50次生成至少应有多个不同值 (碰撞概率极低)
	assert.Greater(t, len(codes), 1, "50次生成的验证码应有多个不同值")
}
