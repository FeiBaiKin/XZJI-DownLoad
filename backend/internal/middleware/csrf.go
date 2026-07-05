package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

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
			SetCSRFCookie(c, GenerateCSRFToken(), 86400)
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

func SecureCookie(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	if strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		return true
	}
	return strings.EqualFold(c.GetHeader("X-Forwarded-Ssl"), "on")
}

func SetCSRFCookie(c *gin.Context, value string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     csrfCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		Secure:   SecureCookie(c),
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
}

// GenerateCSRFToken 生成 32 字节随机 hex 令牌
func GenerateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte("fallback-csrf-token-32-byte-key!"))
	}
	return hex.EncodeToString(b)
}
