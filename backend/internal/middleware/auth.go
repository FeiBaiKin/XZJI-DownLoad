package middleware

import (
	"net/http"
	"strings"

	"xzji/backend/internal/util"

	"github.com/gin-gonic/gin"
)

// AuthRequired JWT 认证中间件
// 优先从 HttpOnly Cookie 读取 token，回退到 Authorization: Bearer header（兼容 API 客户端）
func AuthRequired(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string

		// 优先 HttpOnly Cookie
		if cookie, err := c.Cookie("token"); err == nil && cookie != "" {
			token = cookie
		} else {
			// 回退 Authorization: Bearer header
			auth := c.GetHeader("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				token = strings.TrimPrefix(auth, "Bearer ")
			}
		}

		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未提供认证令牌"})
			return
		}
		claims, err := util.ParseToken(jwtSecret, token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效或过期的令牌"})
			return
		}
		c.Set("userID", claims.ID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// RequireAdmin 管理员权限中间件（需在 AuthRequired 之后使用）
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "无权访问，仅限管理员"})
			return
		}
		c.Next()
	}
}

// GetUserID 从 gin.Context 中取当前用户 ID（便捷方法）
func GetUserID(c *gin.Context) uint {
	id, _ := c.Get("userID")
	if v, ok := id.(uint); ok {
		return v
	}
	return 0
}
