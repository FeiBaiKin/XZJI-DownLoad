package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
)

const csrfCookieName = "csrf_token"
const csrfHeaderName = "X-CSRF-Token"

var safeMethods = map[string]bool{"GET": true, "HEAD": true, "OPTIONS": true}

// CSRF 中间件：double-submit cookie 模式。
// - 首次请求自动下发 CSRF cookie（非 HttpOnly，JS 可读）
// - GET/HEAD/OPTIONS 直接放行
// - POST/PUT/PATCH/DELETE 要求 X-CSRF-Token 请求头与 cookie 值一致
func CSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 确保持有 CSRF cookie（缺失时自动下发）
		if _, err := c.Cookie(csrfCookieName); err != nil {
			c.SetCookie(csrfCookieName, GenerateCSRFToken(), 86400, "/", "", false, false)
		}

		if safeMethods[c.Request.Method] {
			c.Next()
			return
		}

		csrfCookie, _ := c.Cookie(csrfCookieName)
		headerToken := c.GetHeader(csrfHeaderName)
		if headerToken == "" || headerToken != csrfCookie {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "CSRF token 无效"})
			return
		}
		c.Next()
	}
}

// GenerateCSRFToken 生成 32 字节随机 hex 令牌
func GenerateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte("fallback-csrf-token-32-byte-key!"))
	}
	return hex.EncodeToString(b)
}
