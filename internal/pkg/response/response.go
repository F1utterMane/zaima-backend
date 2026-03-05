// Package response 提供统一的 API 响应格式。
// 所有 Handler 的返回均通过此包规范输出，确保前端可以通过固定的
// { code, message, data } 结构解析。
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// R 统一响应体。
type R struct {
	Code    int         `json:"code"`    // 业务状态码: 0=成功, 非0=失败
	Message string      `json:"message"` // 描述信息
	Data    interface{} `json:"data"`    // 业务数据
}

// OK 返回成功响应 (HTTP 200)。
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, R{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// OKWithMsg 返回成功响应并附带自定义消息。
func OKWithMsg(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, R{
		Code:    0,
		Message: msg,
		Data:    data,
	})
}

// Fail 返回失败响应 (HTTP 200，但 code != 0)。
func Fail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, R{
		Code:    code,
		Message: msg,
		Data:    nil,
	})
}

// BadRequest 返回 400 参数错误。
func BadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, R{
		Code:    400,
		Message: msg,
		Data:    nil,
	})
}

// Unauthorized 返回 401 未授权。
func Unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, R{
		Code:    401,
		Message: msg,
		Data:    nil,
	})
}

// ServerError 返回 500 服务内部错误。
func ServerError(c *gin.Context, msg string) {
	c.JSON(http.StatusInternalServerError, R{
		Code:    500,
		Message: msg,
		Data:    nil,
	})
}
