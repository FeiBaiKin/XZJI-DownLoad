package cache

import (
	"time"

	"gorm.io/gorm"
	"xzji/backend/internal/models"
	"xzji/backend/internal/service"
	"xzji/backend/internal/service/payment"
)

// ConfigCache 提供配置缓存功能
type ConfigCache struct {
	emailCache   *TTLCache[string, service.EmailConfig]
	paymentCache *TTLCache[string, payment.PaymentConfig]
	checkinCache *TTLCache[string, models.CheckInConfig]
	levelCache   *TTLCache[string, []models.LevelConfig]
}

// NewConfigCache 创建配置缓存实例
func NewConfigCache() *ConfigCache {
	return &ConfigCache{
		emailCache:   NewTTLCache[string, service.EmailConfig](5 * time.Minute),
		paymentCache: NewTTLCache[string, payment.PaymentConfig](5 * time.Minute),
		checkinCache: NewTTLCache[string, models.CheckInConfig](10 * time.Minute),
		levelCache:   NewTTLCache[string, []models.LevelConfig](10 * time.Minute),
	}
}

// GetEmailConfig 获取邮件配置（带缓存）
func (c *ConfigCache) GetEmailConfig(db *gorm.DB) (service.EmailConfig, error) {
	const key = "email_config"
	
	if cfg, ok := c.emailCache.Get(key); ok {
		return cfg, nil
	}
	
	cfg, err := service.GetEmailConfig(db)
	if err != nil {
		return cfg, err
	}
	
	c.emailCache.Set(key, cfg)
	return cfg, nil
}

// InvalidateEmailConfig 使邮件配置缓存失效
func (c *ConfigCache) InvalidateEmailConfig() {
	c.emailCache.Delete("email_config")
}

// GetPaymentConfig 获取支付配置（带缓存）
func (c *ConfigCache) GetPaymentConfig(db *gorm.DB) (payment.PaymentConfig, error) {
	const key = "payment_config"
	
	if cfg, ok := c.paymentCache.Get(key); ok {
		return cfg, nil
	}
	
	cfg := payment.GetPaymentConfig(db)
	c.paymentCache.Set(key, cfg)
	return cfg, nil
}

// InvalidatePaymentConfig 使支付配置缓存失效
func (c *ConfigCache) InvalidatePaymentConfig() {
	c.paymentCache.Delete("payment_config")
}

// GetCheckinConfig 获取签到配置（带缓存）
func (c *ConfigCache) GetCheckinConfig(db *gorm.DB) (models.CheckInConfig, error) {
	const key = "checkin_config"
	
	if cfg, ok := c.checkinCache.Get(key); ok {
		return cfg, nil
	}
	
	var cfg models.CheckInConfig
	if err := db.First(&cfg).Error; err != nil {
		return cfg, err
	}
	
	// 应用默认值
	if cfg.ConsecutiveDays == 0 {
		cfg.ConsecutiveDays = 7
	}
	if cfg.DailyBalance == 0 {
		cfg.DailyBalance = 0.50
	}
	if cfg.ConsecutiveBonusBalance == 0 {
		cfg.ConsecutiveBonusBalance = 2.00
	}
	if cfg.DailyExp == 0 {
		cfg.DailyExp = 10
	}
	
	c.checkinCache.Set(key, cfg)
	return cfg, nil
}

// InvalidateCheckinConfig 使签到配置缓存失效
func (c *ConfigCache) InvalidateCheckinConfig() {
	c.checkinCache.Delete("checkin_config")
}

// GetLevelConfig 获取等级配置（带缓存）
func (c *ConfigCache) GetLevelConfig(db *gorm.DB) ([]models.LevelConfig, error) {
	const key = "level_config"
	
	if cfgs, ok := c.levelCache.Get(key); ok {
		return cfgs, nil
	}
	
	var cfgs []models.LevelConfig
	if err := db.Order("level ASC").Find(&cfgs).Error; err != nil {
		return nil, err
	}
	
	c.levelCache.Set(key, cfgs)
	return cfgs, nil
}

// InvalidateLevelConfig 使等级配置缓存失效
func (c *ConfigCache) InvalidateLevelConfig() {
	c.levelCache.Delete("level_config")
}

// InvalidateAll 使所有配置缓存失效
func (c *ConfigCache) InvalidateAll() {
	c.emailCache.Clear()
	c.paymentCache.Clear()
	c.checkinCache.Clear()
	c.levelCache.Clear()
}

// Stop 停止缓存清理 goroutine
func (c *ConfigCache) Stop() {
	c.emailCache.Stop()
	c.paymentCache.Stop()
	c.checkinCache.Stop()
	c.levelCache.Stop()
}
