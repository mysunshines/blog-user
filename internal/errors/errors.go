// Package errors 定义 user-service 的应用错误类型。
//
// 设计约定（与「各服务 ErrCode 使用自己 proto 中定义的 enum」架构一致）：
//   - 业务错误码（20001-20016、20100-20102）由各服务的 proto enum 定义，本包仅以常量形式镜像，
//     供 service / repository 构造错误时引用，数值必须与 proto UserErrorCode 对齐。
//   - 通用错误码（10001、10005-10008）由 gocommon 定义并透传，不在此重复声明。
//   - handler 层通过 errors.As 取出 *AppError 的 Code，再映射到 proto 枚举值写入响应。
package errors

import (
	stderrors "errors"
	"fmt"
	"net/http"

	"github.com/mysunshines/gocommon/constants"
)

// Code 是错误码类型，数值与 proto UserErrorCode 及 gocommon 通用码一致。
type Code int

// 通用码透传 gocommon（仅保留本服务实际使用到的 10001、10005-10008），避免散落字面量。
const (
	CodeBadRequest  Code = constants.ErrCodeBadRequest
	CodeInternal    Code = constants.ErrCodeInternal
	CodeServiceDown Code = constants.ErrCodeServiceUnavailable
	CodeTimeout     Code = constants.ErrCodeTimeout
	CodeRateLimited Code = constants.ErrCodeRateLimited
)

// 用户业务错误码（与 proto UserErrorCode 对齐）。
const (
	CodeUserAlreadyExists Code = 20001
	CodeTokenInvalid      Code = 20002
	CodePasswordIncorrect Code = 20003
	CodeInBlacklist       Code = 20004
	CodeUserNotFound      Code = 20005
	CodeTokenExpired      Code = 20006
	CodeUserCreateFailed  Code = 20011
	CodeUserUpdateFailed  Code = 20012
	CodeUserDeleteFailed  Code = 20013
	CodeLoginFailed       Code = 20014
	CodeRegisterFailed    Code = 20015
	CodeUnauthorized      Code = 20100
	CodeForbidden         Code = 20101
	CodeResourceNotFound  Code = 20102
	CodeTooManyRequests   Code = 20016
)

// AppError 是 user-service 的应用错误，携带业务码与可选底层错误，支持 errors.As 断言。
type AppError struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
	HTTP    int    `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// As 是标准库 errors.As 针对 *AppError 的便捷封装，供 handler 层断言应用错误。
func As(err error, target **AppError) bool {
	return stderrors.As(err, target)
}

// New 构造一个 AppError。
func New(code Code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

func BadRequest(message string) *AppError {
	return New(CodeBadRequest, message)
}

func Unauthorized(message string) *AppError {
	return New(CodeUnauthorized, message)
}

func Forbidden(message string) *AppError {
	return New(CodeForbidden, message)
}

func NotFound(message string) *AppError {
	return New(CodeResourceNotFound, message)
}

func Internal(message string, err error) *AppError {
	return &AppError{
		Code:    CodeInternal,
		Message: message,
		Err:     err,
		HTTP:    http.StatusInternalServerError,
	}
}

func UserAlreadyExists(message string) *AppError {
	return New(CodeUserAlreadyExists, message)
}

func TokenInvalid(message string) *AppError {
	return New(CodeTokenInvalid, message)
}

func PasswordIncorrect(message string) *AppError {
	return New(CodePasswordIncorrect, message)
}

func InBlacklist(message string) *AppError {
	return New(CodeInBlacklist, message)
}

func UserNotFound(message string) *AppError {
	return New(CodeUserNotFound, message)
}

func TokenExpired(message string) *AppError {
	return New(CodeTokenExpired, message)
}

func UserCreateFailed(message string) *AppError {
	return New(CodeUserCreateFailed, message)
}

func UserUpdateFailed(message string) *AppError {
	return New(CodeUserUpdateFailed, message)
}

func UserDeleteFailed(message string) *AppError {
	return New(CodeUserDeleteFailed, message)
}

func LoginFailed(message string) *AppError {
	return New(CodeLoginFailed, message)
}

func RegisterFailed(message string) *AppError {
	return New(CodeRegisterFailed, message)
}

func TooManyRequests(message string) *AppError {
	return New(CodeTooManyRequests, message)
}

// IsNotFound 判断 err 链中是否含 NotFound 语义的 AppError。
func IsNotFound(err error) bool {
	var ae *AppError
	if As(err, &ae) {
		return ae.HTTP == http.StatusNotFound
	}
	return false
}

// IsForbidden 判断 err 链中是否含 Forbidden 语义的 AppError。
func IsForbidden(err error) bool {
	var ae *AppError
	if As(err, &ae) {
		return ae.HTTP == http.StatusForbidden
	}
	return false
}
