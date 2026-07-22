package handler

import (
	"log/slog"
	"net/http"

	"xzji/backend/internal/apperr"

	"github.com/gin-gonic/gin"
)

// OK 写出 200 JSON 成功体（保持现有接口契约：直接返回业务对象）。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, data)
}

// Created 写出 201 JSON。
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, data)
}

// Message 写出带 message 字段的成功响应。
func Message(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{"message": msg})
}

// Fail 按 apperr 语义写出错误响应：{error, code}。
// 非 apperr 错误记日志并返回 internal，避免把内部细节泄漏给客户端。
func Fail(c *gin.Context, err error) {
	if err == nil {
		return
	}
	if ae, ok := apperr.As(err); ok {
		if ae.Code == apperr.CodeInternal {
			if ae.Cause != nil {
				slog.Error("internal error", "msg", ae.Message, "err", ae.Cause, "path", c.FullPath())
			} else {
				slog.Error("internal error", "msg", ae.Message, "path", c.FullPath())
			}
		}
		c.JSON(ae.HTTPStatus(), gin.H{
			"error": ae.Message,
			"code":  string(ae.Code),
		})
		return
	}
	slog.Error("untyped error", "err", err, "path", c.FullPath())
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "服务器内部错误",
		"code":  string(apperr.CodeInternal),
	})
}

// FailCode 直接按 code/message 写错误（中间件或参数校验场景）。
func FailCode(c *gin.Context, code apperr.Code, msg string) {
	ae := &apperr.Error{Code: code, Message: msg}
	c.JSON(ae.HTTPStatus(), gin.H{
		"error": msg,
		"code":  string(code),
	})
}

// BindJSON 绑定 JSON，失败时写出 bad_request 并返回 false。
func BindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		Fail(c, apperr.BadRequest("请求参数格式错误"))
		return false
	}
	return true
}
