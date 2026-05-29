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
	Data    interface{} `json:"data,omitempty"`
}

// PageResponse 分页响应
type PageResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Total   int64       `json:"total"`
	Page    int         `json:"page"`
	Size    int         `json:"size"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// PageSuccess 分页成功响应
func PageSuccess(c *gin.Context, data interface{}, total int64, page, size int) {
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

// LoginResponse 登录响应
type LoginResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Token   string      `json:"token"`
	User    interface{} `json:"user"`
}

// SuccessLogin 登录成功响应
func SuccessLogin(c *gin.Context, token string, user interface{}) {
	c.JSON(http.StatusOK, LoginResponse{
		Code:    0,
		Message: "success",
		Token:   token,
		User:    user,
	})
}

// SuccessWithMessage 带消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

// TokenResponse Token验证响应
type TokenResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Valid   bool   `json:"valid"`
	UserID  uint   `json:"user_id"`
}

// SuccessToken Token验证成功响应
func SuccessToken(c *gin.Context, valid bool, userID uint, username string) {
	c.JSON(http.StatusOK, TokenResponse{
		Code:    0,
		Message: "success",
		Valid:   valid,
		UserID:  userID,
	})
}
