package middleware

import (
	"net/http"
	"strings"

	"xzji/backend/internal/models"
	"xzji/backend/internal/util"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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

// RequireAdmin 管理员权限中间件（需在 AuthRequired 之后使用）。
// 优先查库确认当前 role，避免 JWT 内嵌 role 在降权后 24h 内仍可进后台。
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == 0 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "无权访问，仅限管理员"})
			return
		}

		role := ""
		if db := gormDBFromContext(c); db != nil {
			var u models.User
			if err := db.Select("role").First(&u, userID).Error; err == nil {
				role = u.Role
			}
		}
		if role == "" {
			if r, ok := c.Get("role"); ok {
				role, _ = r.(string)
			}
		}
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "无权访问，仅限管理员"})
			return
		}
		c.Set("role", role)
		c.Next()
	}
}

// gormDBFromContext 从 context 中的 app 实例取 *gorm.DB。
// app 以 interface{} 注入，通过小接口避免 middleware→app 循环依赖。
type dbProvider interface {
	GetDB() *gorm.DB
}

func gormDBFromContext(c *gin.Context) *gorm.DB {
	a, ok := c.Get("app")
	if !ok || a == nil {
		return nil
	}
	if p, ok := a.(dbProvider); ok {
		return p.GetDB()
	}
	// *app.App 有导出字段 DB，用类型断言到带 DB 字段的结构不直接可行；
	// 在 app 包增加 GetDB() 方法（见 app.go）。
	return nil
}

// GetUserID 从 gin.Context 中取当前用户 ID（便捷方法）
func GetUserID(c *gin.Context) uint {
	id, _ := c.Get("userID")
	if v, ok := id.(uint); ok {
		return v
	}
	return 0
}
