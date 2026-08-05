package response

import (
	"net/http"

	apperrors "github.com/mysunshines/blog-user/pkg/errors"

	"github.com/gin-gonic/gin"
)

// Response 通用响应
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    any `json:"data,omitempty"`
}

// PageResponse 分页响应
type PageResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    any `json:"data,omitempty"`
	Total   int64       `json:"total"`
	Page    int         `json:"page"`
	Size    int         `json:"size"`
}

// Success 成功响应
func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// PageSuccess 分页成功响应
func PageSuccess(c *gin.Context, data any, total int64, page, size int) {
	c.JSON(http.StatusOK, PageResponse{
		Code:    0,
		Message: "success",
		Data:    data,
		Total:   total,
		Page:    page,
		Size:    size,
	})
}

// Error 错误响应
func Error(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
	})
}

// BadRequest 请求错误
func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, Response{
		Code:    40000,
		Message: message,
	})
}

// Unauthorized 未授权
func Unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, Response{
		Code:    40100,
		Message: message,
	})
}

// Forbidden 禁止访问
func Forbidden(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, Response{
		Code:    40300,
		Message: message,
	})
}

// NotFound 资源不存在
func NotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, Response{
		Code:    40400,
		Message: message,
	})
}

// InternalError 服务器内部错误
func InternalError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, Response{
		Code:    50000,
		Message: message,
	})
}

// FailWithCode 自定义错误码响应
func FailWithCode(c *gin.Context, code int, message string, httpStatus int) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
	})
}

// Fail 错误响应（使用错误对象）
func Fail(c *gin.Context, err error) {
	if appErr, ok := err.(*apperrors.Error); ok {
		c.JSON(http.StatusOK, Response{
			Code:    appErr.Code,
			Message: appErr.Message,
		})
		return
	}
	c.JSON(http.StatusOK, Response{
		Code:    50000,
		Message: err.Error(),
	})
}

// SuccessLogin 登录成功响应（统一使用 data 包裹结构，与 Success/SuccessWithMessage 保持一致）。
// 随响应下发 csrf_token，前端需在非安全方法的已登录请求中原样通过 X-CSRF-Token 头回传。
func SuccessLogin(c *gin.Context, token string, user any, csrfToken string) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"token":      token,
			"user":       user,
			"csrf_token": csrfToken,
		},
	})
}

// SuccessWithMessage 带消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

// SuccessToken Token验证成功响应（统一使用 data 包裹结构，与 Success/SuccessWithMessage 保持一致）
func SuccessToken(c *gin.Context, valid bool, userID uint, username string) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"valid":    valid,
			"user_id":  userID,
			"username": username,
		},
	})
}
