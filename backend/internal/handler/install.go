package handler

import (
	"net/http"
	"strings"

	"xzji/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// InstallStatus GET /api/install/status
func InstallStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"installed": service.IsInstalled()})
}

// Install POST /api/install
func Install(c *gin.Context) {
	// 已安装则拒绝
	if service.IsInstalled() {
		c.JSON(http.StatusForbidden, gin.H{"error": "系统已安装，如需重新安装请删除 .installed 文件"})
		return
	}

	var req service.InstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := service.PerformInstall(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": friendlyInstallError(err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "安装成功！请重启服务后访问首页。",
		"adminUrl": "/login",
	})
}

// friendlyInstallError 将底层错误转为友好提示
func friendlyInstallError(msg string) string {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "connection refused"),
		strings.Contains(lower, "no connection"),
		strings.Contains(lower, "i/o timeout"),
		strings.Contains(lower, "dial tcp"):
		return "无法连接数据库，请检查主机地址和端口"
	case strings.Contains(lower, "access denied"),
		strings.Contains(lower, "authentication"):
		return "数据库认证失败，请检查用户名和密码"
	case strings.Contains(lower, "unknown database"):
		return "数据库不存在且无法创建，请检查账号权限"
	default:
		return msg
	}
}
