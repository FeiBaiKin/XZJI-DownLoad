package handler

import (
	"xzji/backend/internal/app"
	"xzji/backend/internal/database"

	"github.com/gin-gonic/gin"
)

// getApp 从 gin.Context 中获取 App 实例
func getApp(c *gin.Context) *app.App {
	if a, exists := c.Get("app"); exists {
		if appInstance, ok := a.(*app.App); ok {
			return appInstance
		}
	}
	// 回退：如果 context 中没有，使用 database.DB()（兼容旧代码）
	return &app.App{
		DB: database.DB(),
	}
}
