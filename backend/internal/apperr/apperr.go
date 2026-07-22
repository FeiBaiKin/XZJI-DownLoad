// Package apperr 定义业务错误语义：稳定错误码 + 用户可读文案 + HTTP 状态映射。
// handler 层通过 Fail 统一写出 {error, code}，避免静默忽略 err 或随意拼装状态码。
package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

// Code 稳定错误码，前后端可据此分支（勿仅依赖中文 message）。
type Code string

const (
	CodeBadRequest   Code = "bad_request"
	CodeUnauthorized Code = "unauthorized"
	CodeForbidden    Code = "forbidden"
	CodeNotFound     Code = "not_found"
	CodeConflict     Code = "conflict"
	CodeRateLimited  Code = "rate_limited"
	CodeInternal     Code = "internal"
)

// Error 业务错误。
type Error struct {
	Code    Code
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// HTTPStatus 映射 HTTP 状态码。
func (e *Error) HTTPStatus() int {
	if e == nil {
		return http.StatusInternalServerError
	}
	switch e.Code {
	case CodeBadRequest:
		return http.StatusBadRequest
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeRateLimited:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// As 从 error 链中提取 *Error。
func As(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// IsCode 判断错误是否为指定 code。
func IsCode(err error, code Code) bool {
	e, ok := As(err)
	return ok && e.Code == code
}

func BadRequest(msg string) *Error {
	return &Error{Code: CodeBadRequest, Message: msg}
}

func Unauthorized(msg string) *Error {
	return &Error{Code: CodeUnauthorized, Message: msg}
}

func Forbidden(msg string) *Error {
	return &Error{Code: CodeForbidden, Message: msg}
}

func NotFound(msg string) *Error {
	return &Error{Code: CodeNotFound, Message: msg}
}

func Conflict(msg string) *Error {
	return &Error{Code: CodeConflict, Message: msg}
}

func RateLimited(msg string) *Error {
	return &Error{Code: CodeRateLimited, Message: msg}
}

func Internal(msg string) *Error {
	return &Error{Code: CodeInternal, Message: msg}
}

// Wrap 将底层错误包装为 internal，对外暴露安全文案。
func Wrap(msg string, cause error) *Error {
	return &Error{Code: CodeInternal, Message: msg, Cause: cause}
}

// Wrapf 格式化包装。
func Wrapf(cause error, format string, args ...any) *Error {
	return &Error{Code: CodeInternal, Message: fmt.Sprintf(format, args...), Cause: cause}
}
