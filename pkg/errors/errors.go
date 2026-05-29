package errors

import (
	"fmt"
)

// Error 自定义错误
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// NewError 创建错误
func NewError(code int, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// 错误码定义 (区间: 20000-29999, 与 proto/user.proto 中 UserErrorCode 保持一致)
const (
	// 成功
	USER_SUCCESS = 0

	// 业务错误 200xx
	ErrUserAlreadyExists = 20001 // 用户已存在
	ErrTokenInvalid      = 20002 // token无效
	ErrPasswordIncorrect = 20003 // 密码错误
	ErrInBlacklist       = 20004 // 用户在黑名单中
	ErrUserNotFound      = 20005 // 用户未找到
	ErrTokenExpired      = 20006 // token过期

	// 通用错误码
	ErrUnauthorized = 20100 // 未认证
	ErrForbidden    = 20101 // 无权限
	ErrNotFound     = 20102 // 资源不存在

	// 服务器错误 2001x
	ErrInternal       = 20011 // 内部错误
	ErrUpdateFailed   = 20012 // 更新失败
	ErrDeleteFailed   = 20013 // 删除失败
	ErrLoginFailed    = 20014 // 登录失败
	ErrRegisterFailed = 20015 // 注册失败
)

// AppError 是 Error 的别名，保持向后兼容
type AppError = Error

// New 创建错误
func New(code int, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Internal 创建内部错误
func Internal(message string, err error) *Error {
	if err != nil {
		return New(ErrInternal, fmt.Sprintf("%s: %v", message, err))
	}
	return New(ErrInternal, message)
}

// BlacklistFailed 创建黑名单操作失败错误
func BlacklistFailed(err error) *Error {
	return New(ErrInternal, fmt.Sprintf("黑名单操作失败: %v", err))
}
