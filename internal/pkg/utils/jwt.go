// Package utils 提供通用工具函数，包括 JWT 生成/校验、随机码生成等。
package utils

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims 自定义 JWT 声明体。
type JWTClaims struct {
	UserID uint64 `json:"user_id"`
	Phone  string `json:"phone"`
	Role   int    `json:"role"` // 1=老人, 2=年轻人
	jwt.RegisteredClaims
}

// GenerateToken 生成 JWT Token。
//   - userID: 用户主键
//   - phone:  手机号
//   - role:   角色 (1=老人, 2=年轻人)
//   - secret: 签名密钥
//   - hours:  有效时长 (小时)
func GenerateToken(userID uint64, phone string, role int, secret string, hours int) (string, error) {
	claims := JWTClaims{
		UserID: userID,
		Phone:  phone,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(hours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "zaima-backend",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken 解析并验证 JWT Token，返回自定义声明体。
func ParseToken(tokenStr, secret string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		// 【安全】防止 JWT 算法混淆攻击 (Algorithm Confusion)
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("无效的 Token")
}

// GenerateSMSCode 生成6位数字短信验证码。
func GenerateSMSCode() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	code := (int(b[0])<<16 | int(b[1])<<8 | int(b[2])) % 1000000
	return fmt.Sprintf("%06d", code)
}
