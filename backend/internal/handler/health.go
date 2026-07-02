package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Health GET /api/health
func Health(c *gin.Context) {
	status := "disconnected"
	app := getApp(c)
	if app != nil && app.DB != nil {
		sqlDB, err := app.DB.DB()
		if err == nil && sqlDB != nil {
			if err := sqlDB.Ping(); err == nil {
				status = "connected"
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "xzji-backend",
		"database":  status,
	})
}
