package database

import (
	"log/slog"

	"xzji/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Seed 写入默认数据（仅当表为空时插入，幂等）
func Seed(db *gorm.DB) error {
	if err := seedDefaultPages(db); err != nil {
		return err
	}
	if err := seedCheckInConfig(db); err != nil {
		return err
	}
	if err := seedLevelConfig(db); err != nil {
		return err
	}
	if err := seedMembershipConfig(db); err != nil {
		return err
	}
	if err := seedDefaultKeywords(db); err != nil {
		return err
	}
	if err := seedDefaultEmailSettings(db); err != nil {
		return err
	}
	return nil
}

func seedDefaultPages(db *gorm.DB) error {
	var count int64
	db.Model(&models.Page{}).Count(&count)
	if count > 0 {
		return nil
	}
	pages := []models.Page{
		{Title: "隐私协议", Slug: "privacy", Content: "<h2>隐私协议</h2><p>请在此编辑隐私协议内容。</p>", IsPublished: true},
		{Title: "用户协议", Slug: "terms", Content: "<h2>用户协议</h2><p>请在此编辑用户协议内容。</p>", IsPublished: true},
		{Title: "关于我们", Slug: "about", Content: "<h2>关于我们</h2><p>请在此编辑关于我们页面的内容。</p>", IsPublished: true},
	}
	if err := db.Create(&pages).Error; err != nil {
		return err
	}
	slog.Info("已插入默认页面", "count", len(pages))
	return nil
}

func seedCheckInConfig(db *gorm.DB) error {
	var count int64
	db.Model(&models.CheckInConfig{}).Count(&count)
	if count > 0 {
		return nil
	}
	cfg := models.CheckInConfig{
		DailyBalance:            0.50,
		ConsecutiveDays:         7,
		ConsecutiveBonusBalance: 2.00,
		DailyExp:                10,
	}
	if err := db.Create(&cfg).Error; err != nil {
		return err
	}
	slog.Info("已插入默认签到配置")
	return nil
}

func seedLevelConfig(db *gorm.DB) error {
	var count int64
	db.Model(&models.LevelConfig{}).Count(&count)
	if count > 0 {
		return nil
	}
	levels := []models.LevelConfig{
		{Level: 1, ExpRequired: 0, Label: "初来乍到"},
		{Level: 2, ExpRequired: 100, Label: "见习会员"},
		{Level: 3, ExpRequired: 300, Label: "正式会员"},
		{Level: 4, ExpRequired: 600, Label: "活跃会员"},
		{Level: 5, ExpRequired: 1000, Label: "资深会员"},
		{Level: 6, ExpRequired: 1500, Label: "高级会员"},
		{Level: 7, ExpRequired: 2200, Label: "黄金会员"},
		{Level: 8, ExpRequired: 3200, Label: "铂金会员"},
		{Level: 9, ExpRequired: 4500, Label: "钻石会员"},
		{Level: 10, ExpRequired: 6000, Label: "至尊会员"},
	}
	if err := db.Create(&levels).Error; err != nil {
		return err
	}
	slog.Info("已插入默认等级配置", "count", len(levels))
	return nil
}

func seedMembershipConfig(db *gorm.DB) error {
	var count int64
	db.Model(&models.MembershipConfig{}).Count(&count)
	if count > 0 {
		return nil
	}
	memberships := []models.MembershipConfig{
		{LevelName: "基础会员", Days: 30, Price: 30.00, DailyDownloadLimit: 20, Discount: 0.90, Color: "#888888"},
		{LevelName: "高级会员", Days: 90, Price: 80.00, DailyDownloadLimit: 50, Discount: 0.80, Color: "#c9a44b"},
		{LevelName: "至尊会员", Days: 365, Price: 200.00, DailyDownloadLimit: -1, Discount: 0.60, Color: "#e03e3e"},
	}
	if err := db.Create(&memberships).Error; err != nil {
		return err
	}
	slog.Info("已插入默认会员配置", "count", len(memberships))
	return nil
}

func seedDefaultKeywords(db *gorm.DB) error {
	keywords := []models.KeywordResponse{
		{Keyword: "hello", Response: "你好！欢迎使用关键词搜索应用。"},
		{Keyword: "node", Response: "Node.js 是一个基于 Chrome V8 引擎的 JavaScript 运行时。"},
		{Keyword: "react", Response: "React 是一个用于构建用户界面的 JavaScript 库。"},
		{Keyword: "mysql", Response: "MySQL 是最流行的关系型数据库管理系统之一。"},
		{Keyword: "search", Response: "搜索功能已启用，正在探索关键词的奥秘。"},
	}
	// INSERT IGNORE 语义：用 clause.OnConflict 忽略唯一键冲突
	for _, k := range keywords {
		if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&k).Error; err != nil {
			return err
		}
	}
	slog.Info("已插入默认关键词", "count", len(keywords))
	return nil
}

func seedDefaultEmailSettings(db *gorm.DB) error {
	// 仅在不存在 email_* 配置时插入默认值
	var count int64
	db.Model(&models.SiteSetting{}).Where("setting_key LIKE ?", "email_%").Count(&count)
	if count > 0 {
		return nil
	}
	defaults := []models.SiteSetting{
		{SettingKey: "email_smtp_host", SettingValue: ""},
		{SettingKey: "email_smtp_port", SettingValue: "465"},
		{SettingKey: "email_smtp_user", SettingValue: ""},
		{SettingKey: "email_smtp_pass", SettingValue: ""},
		{SettingKey: "email_from_name", SettingValue: "系统通知"},
		{SettingKey: "email_force_verify", SettingValue: "0"},
		{SettingKey: "email_enable_passwordless", SettingValue: "0"},
	}
	for _, s := range defaults {
		if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&s).Error; err != nil {
			return err
		}
	}
	slog.Info("已插入默认邮件配置", "count", len(defaults))
	return nil
}
