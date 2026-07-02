package service

import (
	"fmt"
	"strings"
	"time"

	"xzji/backend/internal/config"
	"xzji/backend/internal/database"
	"xzji/backend/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// InstallRequest 安装向导提交参数
type InstallRequest struct {
	SiteName        string `json:"siteName"`
	SiteDescription string `json:"siteDescription"`
	SiteURL         string `json:"siteURL"`
	FooterText      string `json:"footerText"`
	AdminUsername   string `json:"adminUsername"`
	AdminPassword   string `json:"adminPassword"`
	AdminEmail      string `json:"adminEmail"`
	DBHost          string `json:"dbHost"`
	DBPort          int    `json:"dbPort"`
	DBUser          string `json:"dbUser"`
	DBPassword      string `json:"dbPassword"`
	DBName          string `json:"dbName"`
}

// Validate 校验必填项
func (r *InstallRequest) Validate() error {
	if r.SiteName == "" {
		return fmt.Errorf("网站名称不能为空")
	}
	if r.SiteURL == "" {
		return fmt.Errorf("站点地址不能为空")
	}
	// 基本的 URL 校验：必须以 http:// 或 https:// 开头
	if !strings.HasPrefix(r.SiteURL, "http://") && !strings.HasPrefix(r.SiteURL, "https://") {
		return fmt.Errorf("站点地址必须以 http:// 或 https:// 开头")
	}
	if len(r.AdminUsername) < 3 {
		return fmt.Errorf("管理员账号至少 3 个字符")
	}
	if len(r.AdminPassword) < 6 {
		return fmt.Errorf("管理员密码至少 6 个字符")
	}
	if r.DBHost == "" {
		return fmt.Errorf("数据库地址不能为空")
	}
	if r.DBUser == "" {
		return fmt.Errorf("数据库用户名不能为空")
	}
	if r.DBName == "" {
		r.DBName = "xzji"
	}
	if r.DBPort == 0 {
		r.DBPort = 3306
	}
	return nil
}

// PerformInstall 执行安装全流程
func PerformInstall(req *InstallRequest) error {
	// 1. 建库 + 建表 + 种子
	if err := database.SetupFromInstall(req.DBHost, req.DBPort, req.DBUser, req.DBPassword, req.DBName); err != nil {
		return err
	}

	// 2. 连接到新库写入管理员与站点配置
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		req.DBUser, req.DBPassword, req.DBHost, req.DBPort, req.DBName)
	gdb, err := gorm.Open(mysqlOpen(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("连接新库失败: %w", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	// 3. 创建管理员（bcrypt 成本因子 10，与原 bcryptjs 兼容）
	if err := createAdmin(gdb, req.AdminUsername, req.AdminPassword, req.AdminEmail); err != nil {
		return fmt.Errorf("创建管理员失败: %w", err)
	}

	// 4. 写入站点设置
	if err := writeSiteSettings(gdb, req); err != nil {
		return fmt.Errorf("写入站点设置失败: %w", err)
	}

	// 5. 写 config.yaml + .installed 锁
	jwtSecret := config.GenerateJWTSecret()
	allowedOrigins := []string{req.SiteURL}
	if err := config.SaveConfig(req.DBHost, req.DBPort, req.DBUser, req.DBPassword, req.DBName, allowedOrigins, jwtSecret); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	if err := WriteInstallLock(); err != nil {
		return fmt.Errorf("写入安装锁失败: %w", err)
	}

	return nil
}

func createAdmin(db *gorm.DB, username, password, email string) error {
	// 若管理员已存在则跳过（幂等，对应原 ON DUPLICATE KEY 语义）
	var exist int64
	db.Model(&models.User{}).Where("username = ?", username).Count(&exist)
	if exist > 0 {
		return nil
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return err
	}
	admin := models.User{
		Username: username,
		Password: string(hashed),
		Role:     "admin",
		Email:    email,
		Nickname: username,
		Balance:  0,
		Exp:      0,
		Level:    1,
	}
	return db.Create(&admin).Error
}

func writeSiteSettings(db *gorm.DB, req *InstallRequest) error {
	desc := req.SiteDescription
	if desc == "" {
		desc = "输入关键词，发现后端配置的响应内容"
	}
	footer := req.FooterText
	if footer == "" {
		footer = "Keyword Search App"
	}
	settings := []models.SiteSetting{
		{SettingKey: "siteName", SettingValue: req.SiteName},
		{SettingKey: "siteDescription", SettingValue: desc},
		{SettingKey: "footerText", SettingValue: footer},
		{SettingKey: "siteLogoType", SettingValue: "text"},
		{SettingKey: "siteLogoUrl", SettingValue: ""},
	}
	for _, s := range settings {
		// INSERT ... ON DUPLICATE KEY UPDATE
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "setting_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"setting_value"}),
		}).Create(&s).Error; err != nil {
			return err
		}
	}
	return nil
}

// mysqlOpen 包一层便于测试时替换；实际就是 gorm mysql.Open
// 这里直接返回 driver，避免在本文件引入循环依赖
func mysqlOpen(dsn string) gorm.Dialector {
	return mysqlDialector(dsn)
}

// IsInstalled 安装锁检测
func IsInstalled() bool {
	return installLockExists()
}

// WriteInstallLock 写入安装锁文件
func WriteInstallLock() error {
	return writeInstallLockImpl(time.Now())
}
