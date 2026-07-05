package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// AllowedOrigins CORS 允许的来源列表，由 main.go 从配置注入。
// 为空时回退到 "*"（开发环境），生产环境应配置具体域名。
var AllowedOrigins []string

// CORS 跨域中间件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowOrigin := ""

		if len(AllowedOrigins) == 0 {
			// 未配置：开发环境允许所有来源；有 Origin 时回显以兼容 credentials。
			if origin != "" {
				allowOrigin = origin
			} else {
				allowOrigin = "*"
			}
		} else if origin != "" {
			// 检查请求来源是否在白名单中
			for _, allowed := range AllowedOrigins {
				if strings.EqualFold(origin, allowed) {
					allowOrigin = origin
					break
				}
			}
		}

		if allowOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowOrigin)
			c.Header("Vary", "Origin")
			if allowOrigin != "*" {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With, X-CSRF-Token")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
