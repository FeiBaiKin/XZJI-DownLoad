package handler

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"xzji/backend/internal/config"
	"xzji/backend/internal/license"

	"github.com/gin-gonic/gin"
)

// getLicense 获取授权管理器；未初始化（理论上仅安装模式）返回 nil 并已写响应
func getLicense(c *gin.Context) *license.Manager {
	m := getApp(c).License
	if m == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "授权模块未初始化"})
	}
	return m
}

// PublicLicenseStatus GET /api/license/status
// 前台判断是否显示停服遮罩，只暴露有效性与统一文案，不泄露授权细节。
func PublicLicenseStatus(c *gin.Context) {
	m := getApp(c).License
	if m == nil || m.Valid() {
		c.JSON(http.StatusOK, gin.H{"valid": true})
		return
	}
	c.JSON(http.StatusOK, gin.H{"valid": false, "message": license.BlockedMessage})
}

// AdminLicenseGet GET /api/admin/license —— 授权状态快照
func AdminLicenseGet(c *gin.Context) {
	m := getLicense(c)
	if m == nil {
		return
	}
	c.JSON(http.StatusOK, m.Snapshot())
}

// AdminLicenseActivate POST /api/admin/license/activate {code}
//
// 域名自动获取：从请求头探测本站真实域名（Origin 优先，回退 Host），
// 归一化后作为授权上报域名，并在激活成功时回写 config.yaml 的
// server.allowed_origins，使重启后域名仍可从配置加载。
func AdminLicenseActivate(c *gin.Context) {
	m := getLicense(c)
	if m == nil {
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Code) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入授权码"})
		return
	}

	// 探测本站域名：浏览器跨域请求带 Origin（含协议），同源请求可能无 Origin，回退 Host
	origin := c.Request.Header.Get("Origin")
	if origin == "" {
		// Host 不含协议，按 https 补全以便复用归一化逻辑（协议会被剥离，仅影响端口判断）
		if h := c.Request.Host; h != "" {
			scheme := "https"
			if c.Request.TLS == nil && (c.Request.Header.Get("X-Forwarded-Proto") == "http" || strings.HasPrefix(h, "localhost")) {
				scheme = "http"
			}
			origin = scheme + "://" + h
		}
	}
	domain := license.NormalizeDomain(origin)

	sr, err := m.Activate(req.Code, domain)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 激活成功（active 或同域名幂等）时回写配置，持久化探测到的站点地址
	if origin != "" && sr.Status == license.StatusActive {
		if err := config.SetAllowedOrigins([]string{origin}); err != nil {
			slog.Warn("回写 allowed_origins 失败（不影响本次激活）", "err", err, "origin", origin)
		}
	}

	msg := sr.Message
	if msg == "" {
		switch sr.Status {
		case license.StatusActive:
			msg = "授权有效"
		case license.StatusExpired:
			msg = "该授权码已过期"
		case license.StatusRevoked:
			msg = "该授权码已被吊销"
		case license.StatusDomainMismatch:
			msg = "该授权码已绑定其他域名"
		default:
			msg = "授权状态：" + sr.Status
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": msg, "status": sr.Status, "snapshot": m.Snapshot()})
}

// AdminLicenseHeartbeat POST /api/admin/license/heartbeat —— 手动立即心跳
func AdminLicenseHeartbeat(c *gin.Context) {
	m := getLicense(c)
	if m == nil {
		return
	}
	if _, err := m.Heartbeat(); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "snapshot": m.Snapshot()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "心跳成功，授权状态已刷新", "snapshot": m.Snapshot()})
}

// AdminLicenseUpgrade POST /api/admin/license/upgrade
// 下载并应用最新版本；应用成功后延迟自重启（先让本响应发出）。
func AdminLicenseUpgrade(c *gin.Context) {
	m := getLicense(c)
	if m == nil {
		return
	}
	res, err := m.Upgrade()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
	if res.Restarting {
		m.ScheduleRestart(1500 * time.Millisecond)
	}
}
