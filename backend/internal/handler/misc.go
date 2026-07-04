package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"xzji/backend/internal/middleware"
	"xzji/backend/internal/models"
	"xzji/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// === 签到 ===

func CheckInStatus(c *gin.Context) {
	res, _ := service.CheckInStatus(getApp(c).DB, middleware.GetUserID(c))
	c.JSON(http.StatusOK, res)
}

func CheckIn(c *gin.Context) {
	res, err := service.CheckIn(getApp(c).DB, middleware.GetUserID(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// === 等级 ===

func LevelInfo(c *gin.Context) {
	res, _ := service.LevelInfo(getApp(c).DB, middleware.GetUserID(c))
	c.JSON(http.StatusOK, res)
}

// === 仪表盘 ===

func AdminStats(c *gin.Context) {
	res, _ := service.DashboardStats(getApp(c).DB)
	c.JSON(http.StatusOK, res)
}

// === 站点设置 ===

// PublicSettings GET /api/settings
// 公开站点设置（仅站点相关键，屏蔽 payment_*/email_* 等敏感配置）。
// 对齐原项目 /api/settings 供首页读取 siteName/siteDescription/footerText/siteLogo*。
func PublicSettings(c *gin.Context) {
	keys := []string{"siteName", "siteDescription", "footerText", "siteLogoType", "siteLogoUrl", "siteIcp", "sitePolice"}
	var rows []models.SiteSetting
	getApp(c).DB.Where("setting_key IN ?", keys).Find(&rows)
	m := map[string]string{
		"siteName":        "关键词搜索",
		"siteDescription": "输入关键词，发现后端配置的响应内容",
		"footerText":      "Keyword Search App",
		"siteLogoType":    "text",
		"siteLogoUrl":     "",
		"siteIcp":         "",
		"sitePolice":      "",
	}
	for _, r := range rows {
		m[r.SettingKey] = r.SettingValue
	}
	c.JSON(http.StatusOK, m)
}

// PublicSiteConfig GET /api/site/config
// 公开站点配置（非敏感），供登录/注册页判断强制验证与免密登录开关。
func PublicSiteConfig(c *gin.Context) {
	app := getApp(c)
	cfg, err := app.Cache.GetEmailConfig(app.DB)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"forceVerify": false, "enablePasswordless": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"forceVerify":         cfg.ForceVerify,
		"enablePasswordless":  cfg.EnablePasswordless,
	})
}

func AdminSettingsGet(c *gin.Context) {
	res, _ := service.SiteSettings(getApp(c).DB)
	c.JSON(http.StatusOK, res)
}

func AdminSettingsPut(c *gin.Context) {
	var s map[string]string
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := service.UpdateSiteSettings(getApp(c).DB, s); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "保存成功"})
}

// === 邮件配置 ===

func AdminEmailConfigGet(c *gin.Context) {
	app := getApp(c)
	cfg, _ := app.Cache.GetEmailConfig(app.DB)
	c.JSON(http.StatusOK, gin.H{
		"smtpHost": cfg.SMTPHost, "smtpPort": cfg.SMTPPort, "smtpUser": cfg.SMTPUser,
		"smtpPass": cfg.SMTPPass, "fromName": cfg.FromName,
		"forceVerify": cfg.ForceVerify, "enablePasswordless": cfg.EnablePasswordless,
	})
}

func AdminEmailConfigPut(c *gin.Context) {
	var s map[string]string
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	// 前端发送 email_* 键，原样存储（GetEmailConfig 读取 email_* 前缀）
	settings := map[string]string{}
	for k, v := range s {
		settings[k] = v
	}
	app := getApp(c)
	if err := service.UpdateEmailConfig(app.DB, settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	// 使缓存失效
	app.Cache.InvalidateEmailConfig()
	c.JSON(http.StatusOK, gin.H{"message": "保存成功"})
}

func AdminEmailTest(c *gin.Context) {
	var req struct{ To string `json:"to"` }
	if err := c.ShouldBindJSON(&req); err != nil || req.To == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供测试邮箱"})
		return
	}
	if err := service.EmailTest(getApp(c).DB, req.To); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "测试邮件已发送"})
}

// TriggerMembershipReminders POST /api/admin/trigger-membership-reminders
// 手动触发会员到期提醒检查（对齐原项目 admin trigger 端点）。
func TriggerMembershipReminders(c *gin.Context) {
	sent, err := service.CheckMembershipExpiryReminders(getApp(c).DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "触发失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "会员到期提醒已触发", "sent": sent})
}

// === 签到/等级/会员 配置 (admin) ===

func AdminCheckinConfigGet(c *gin.Context) {
	app := getApp(c)
	cfg, _ := app.Cache.GetCheckinConfig(app.DB)
	c.JSON(http.StatusOK, gin.H{
		"daily_balance":             cfg.DailyBalance,
		"consecutive_days":          cfg.ConsecutiveDays,
		"consecutive_bonus_balance": cfg.ConsecutiveBonusBalance,
		"daily_exp":                 cfg.DailyExp,
	})
}

func AdminCheckinConfigPut(c *gin.Context) {
	var s map[string]any
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	app := getApp(c)
	app.DB.Exec("UPDATE check_in_config SET daily_balance=?, consecutive_days=?, consecutive_bonus_balance=?, daily_exp=? WHERE id=1",
		s["daily_balance"], s["consecutive_days"], s["consecutive_bonus_balance"], s["daily_exp"])
	// 使缓存失效
	app.Cache.InvalidateCheckinConfig()
	c.JSON(http.StatusOK, gin.H{"message": "保存成功"})
}

func AdminLevelConfigGet(c *gin.Context) {
	app := getApp(c)
	cfgs, _ := app.Cache.GetLevelConfig(app.DB)
	list := make([]map[string]any, len(cfgs))
	for i, cfg := range cfgs {
		list[i] = map[string]any{
			"level":        cfg.Level,
			"exp_required": cfg.ExpRequired,
			"label":        cfg.Label,
		}
	}
	c.JSON(http.StatusOK, list)
}

func AdminLevelConfigPut(c *gin.Context) {
	var items []map[string]any
	if err := c.ShouldBindJSON(&items); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	// 全量同步：按 level upsert，删除不在列表中的等级（对齐原版可增删行为）
	app := getApp(c)
	db := app.DB
	keep := make([]any, 0, len(items))
	for _, it := range items {
		db.Exec("INSERT INTO level_config (level, exp_required, label, updated_at) VALUES (?, ?, ?, NOW()) ON DUPLICATE KEY UPDATE exp_required = VALUES(exp_required), label = VALUES(label), updated_at = NOW()",
			it["level"], it["exp_required"], it["label"])
		keep = append(keep, it["level"])
	}
	if len(keep) > 0 {
		db.Where("level NOT IN ?", keep).Delete(&models.LevelConfig{})
	} else {
		db.Where("1 = 1").Delete(&models.LevelConfig{})
	}
	// 使缓存失效
	app.Cache.InvalidateLevelConfig()
	c.JSON(http.StatusOK, gin.H{"message": "保存成功"})
}

func AdminMembershipConfigGet(c *gin.Context) {
	var list []map[string]any
	getApp(c).DB.Raw("SELECT level_name, days, price, daily_download_limit, discount, color FROM membership_config ORDER BY price").Scan(&list)
	c.JSON(http.StatusOK, list)
}

func AdminMembershipConfigPut(c *gin.Context) {
	var items []map[string]any
	if err := c.ShouldBindJSON(&items); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	// 全量同步：按 level_name upsert，删除不在列表中的会员等级
	db := getApp(c).DB
	keep := make([]any, 0, len(items))
	for _, it := range items {
		db.Exec("INSERT INTO membership_config (level_name, days, price, daily_download_limit, discount, color, updated_at) VALUES (?, ?, ?, ?, ?, ?, NOW()) ON DUPLICATE KEY UPDATE days = VALUES(days), price = VALUES(price), daily_download_limit = VALUES(daily_download_limit), discount = VALUES(discount), color = VALUES(color), updated_at = NOW()",
			it["level_name"], it["days"], it["price"], it["daily_download_limit"], it["discount"], it["color"])
		keep = append(keep, it["level_name"])
	}
	if len(keep) > 0 {
		db.Where("level_name NOT IN ?", keep).Delete(&models.MembershipConfig{})
	} else {
		db.Where("1 = 1").Delete(&models.MembershipConfig{})
	}
	c.JSON(http.StatusOK, gin.H{"message": "保存成功"})
}

func AdminPaymentConfigGet(c *gin.Context) {
	app := getApp(c)
	cfg, _ := app.Cache.GetPaymentConfig(app.DB)

	// 将 EnabledProviders 数组序列化为 JSON 字符串，与前端契约一致
	enabledProvidersJSON := "[]"
	if len(cfg.EnabledProviders) > 0 {
		if b, err := json.Marshal(cfg.EnabledProviders); err == nil {
			enabledProvidersJSON = string(b)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"payment_site_url":            cfg.SiteURL,
		"payment_enabled_providers":   enabledProvidersJSON,
		"payment_wx_appid":            cfg.WxAppId,
		"payment_wx_mchid":            cfg.WxMchId,
		"payment_wx_serial_no":        cfg.WxSerialNo,
		"payment_wx_api_v3_key":       cfg.WxApiV3Key,
		"payment_wx_private_key":      cfg.WxPrivateKey,
		"payment_wx_notify_url":       cfg.WxNotifyUrl,
		"payment_ali_appid":           cfg.AliAppId,
		"payment_ali_private_key":     cfg.AliPrivateKey,
		"payment_ali_public_key":      cfg.AliPublicKey,
		"payment_ali_scene":           cfg.AliScene,
		"payment_ali_notify_url":      cfg.AliNotifyUrl,
		"payment_ali_return_url":      cfg.AliReturnUrl,
		"payment_epay_gateway":        cfg.EpayGateway,
		"payment_epay_pid":            cfg.EpayPid,
		"payment_epay_key":            cfg.EpayKey,
		"payment_epay_notify_url":     cfg.EpayNotifyUrl,
		"payment_epay_return_url":     cfg.EpayReturnUrl,
	})
}

func AdminPaymentConfigPut(c *gin.Context) {
	var s map[string]string
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	// 前端发送 payment_* 键，原样存储（与 GET 返回的 payment_* 一致）
	settings := map[string]string{}
	for k, v := range s {
		settings[k] = v
	}
	app := getApp(c)
	if err := service.UpdateSiteSettings(app.DB, settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	// 使缓存失效
	app.Cache.InvalidatePaymentConfig()
	c.JSON(http.StatusOK, gin.H{"message": "保存成功"})
}

// === 用户管理 (admin) ===

func AdminUsersList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "15"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 15
	}
	var total int64
	getApp(c).DB.Model(&models.User{}).Count(&total)
	var users []models.User
	getApp(c).DB.Order("id ASC").Offset((page - 1) * limit).Limit(limit).Find(&users)
	c.JSON(http.StatusOK, gin.H{"users": users, "total": total})
}

func AdminUserGet(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var u map[string]any
	getApp(c).DB.Raw("SELECT * FROM users WHERE id = ?", id).Scan(&u)
	if len(u) == 0 { c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"}); return }
	c.JSON(http.StatusOK, u)
}

func AdminUserUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var s map[string]any
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	allowed := map[string]any{}
	if v, ok := s["nickname"]; ok { allowed["nickname"] = v }
	if v, ok := s["email"]; ok { allowed["email"] = v }
	if v, ok := s["role"]; ok { allowed["role"] = v }
	if v, ok := s["balance"]; ok { allowed["balance"] = v }
	if v, ok := s["exp"]; ok { allowed["exp"] = v }
	if v, ok := s["level"]; ok { allowed["level"] = v }
	// 会员字段（编辑用户时设置/清除会员）
	ml, _ := s["membership_level"].(string)
	mea, _ := s["membership_expires_at"].(string)
	if ml == "" {
		allowed["membership_level"] = nil
		allowed["membership_expires_at"] = nil
	} else {
		allowed["membership_level"] = ml
		if mea != "" {
			// datetime-local "2026-07-29T12:00" -> "2026-07-29 12:00:00"
			t := strings.ReplaceAll(mea, "T", " ")
			if len(t) == 16 {
				t += ":00"
			}
			allowed["membership_expires_at"] = t
		}
	}
	if len(allowed) == 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "无有效更新字段"}); return }
	getApp(c).DB.Model(&models.User{}).Where("id = ?", id).Updates(allowed)
	c.JSON(http.StatusOK, gin.H{"message": "已更新"})
}

func AdminUserDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	getApp(c).DB.Delete(&models.User{}, id)
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}
