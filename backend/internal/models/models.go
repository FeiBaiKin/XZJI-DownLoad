package models

import (
	"time"

	"gorm.io/gorm"
)

// User 用户表（对齐原 users 表）
type User struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	Username            string     `gorm:"size:50;uniqueIndex" json:"username"`
	Password            string     `gorm:"size:255" json:"-"`
	Role                string     `gorm:"size:20;default:user" json:"role"`
	Email               string     `gorm:"size:100;default:''" json:"email"`
	Nickname            string     `gorm:"size:50;default:''" json:"nickname"`
	Avatar              string     `gorm:"size:255;default:''" json:"avatar"`
	Balance             float64    `gorm:"type:decimal(10,2);default:0" json:"balance"`
	Exp                 int        `gorm:"default:0" json:"exp"`
	Level               int        `gorm:"default:1" json:"level"`
	MembershipLevel     *string    `gorm:"size:50" json:"membership_level"`
	MembershipExpiresAt *time.Time `json:"membership_expires_at"`
	DailyDownloads      int        `gorm:"default:0" json:"daily_downloads"`
	DownloadDate        *time.Time `gorm:"type:date" json:"download_date"`
	EmailVerified       bool       `gorm:"default:false" json:"email_verified"`
	CreatedAt           time.Time  `json:"created_at"`
}

func (User) TableName() string { return "users" }

// 关键词资源状态
const (
	KeywordStatusDraft     = "draft"     // 草稿 / 下架（前台不可见）
	KeywordStatusPublished = "published" // 已上架
)

// KeywordResponse 关键词响应（对齐 keyword_responses）
type KeywordResponse struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Keyword    string    `gorm:"size:255;uniqueIndex" json:"keyword"`
	Response   string    `gorm:"type:text" json:"response"`
	Icon       string    `gorm:"size:255;default:''" json:"icon"`
	Attributes string    `gorm:"type:text;default:''" json:"attributes"` // JSON: [{label,value}]
	// Status: draft | published。默认 published，兼容历史数据。
	Status    string    `gorm:"size:20;default:published;index" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (KeywordResponse) TableName() string { return "keyword_responses" }

// KeywordCategory 关键词分类（对齐 keyword_categories）
type KeywordCategory struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:100" json:"name"`
	Description string    `gorm:"size:500;default:''" json:"description"`
	SortOrder   int       `gorm:"default:0" json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (KeywordCategory) TableName() string { return "keyword_categories" }

// KeywordCategoryLink 关键词←→分类 多对多关联（对齐 keyword_category_links）
type KeywordCategoryLink struct {
	ID         uint            `gorm:"primaryKey" json:"id"`
	KeywordID  uint            `gorm:"uniqueIndex:unique_link" json:"keyword_id"`
	Keyword    KeywordResponse `gorm:"foreignKey:KeywordID;constraint:OnDelete:CASCADE" json:"-"`
	CategoryID uint            `gorm:"uniqueIndex:unique_link" json:"category_id"`
	Category   KeywordCategory `gorm:"foreignKey:CategoryID;constraint:OnDelete:CASCADE" json:"-"`
}

func (KeywordCategoryLink) TableName() string { return "keyword_category_links" }

// Page 自定义页面（对齐 pages）
type Page struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Title       string    `gorm:"size:200" json:"title"`
	Slug        string    `gorm:"size:100;uniqueIndex" json:"slug"`
	Content     string    `gorm:"type:text" json:"content"`
	IsPublished bool      `gorm:"default:false" json:"is_published"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Page) TableName() string { return "pages" }

// Announcement 公告（对齐 announcements）
type Announcement struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"size:200" json:"title"`
	Content   string    `gorm:"type:text" json:"content"`
	Type      string    `gorm:"size:20;default:info" json:"type"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Announcement) TableName() string { return "announcements" }

// SiteSetting KV 配置表（对齐 site_settings）
type SiteSetting struct {
	SettingKey   string `gorm:"primaryKey;size:100" json:"setting_key"`
	SettingValue string `gorm:"type:text" json:"setting_value"`
}

func (SiteSetting) TableName() string { return "site_settings" }

// CheckInConfig 签到配置（对齐 check_in_config）
type CheckInConfig struct {
	ID                      uint      `gorm:"primaryKey" json:"id"`
	DailyBalance            float64   `gorm:"type:decimal(10,2);default:0.50" json:"daily_balance"`
	ConsecutiveDays         int       `gorm:"default:7" json:"consecutive_days"`
	ConsecutiveBonusBalance float64   `gorm:"type:decimal(10,2);default:2.00" json:"consecutive_bonus_balance"`
	DailyExp                int       `gorm:"default:10" json:"daily_exp"`
	UpdatedAt               time.Time `json:"updated_at"`
}

func (CheckInConfig) TableName() string { return "check_in_config" }

// CheckInRecord 签到记录（对齐 check_in_records）
type CheckInRecord struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"uniqueIndex:unique_daily" json:"user_id"`
	CheckInDate   time.Time `gorm:"type:date;uniqueIndex:unique_daily" json:"check_in_date"`
	BalanceEarned float64   `gorm:"type:decimal(10,2);default:0.00" json:"balance_earned"`
	ExpEarned     int       `gorm:"default:0" json:"exp_earned"`
	IsBonus       bool      `gorm:"default:false" json:"is_bonus"`
	CreatedAt     time.Time `json:"created_at"`
}

func (CheckInRecord) TableName() string { return "check_in_records" }

// LevelConfig 等级配置（对齐 level_config）
type LevelConfig struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Level       int       `gorm:"uniqueIndex" json:"level"`
	ExpRequired int       `json:"exp_required"`
	Label       string    `gorm:"size:50;default:''" json:"label"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (LevelConfig) TableName() string { return "level_config" }

// MembershipConfig 会员配置（对齐 membership_config）
type MembershipConfig struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	LevelName          string    `gorm:"size:50;uniqueIndex" json:"level_name"`
	Days               int       `json:"days"`
	Price              float64   `gorm:"type:decimal(10,2)" json:"price"`
	DailyDownloadLimit int       `gorm:"default:20" json:"daily_download_limit"`
	Discount           float64   `gorm:"type:decimal(5,2);default:0" json:"discount"`
	Color              string    `gorm:"size:20;default:''" json:"color"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (MembershipConfig) TableName() string { return "membership_config" }

// MembershipOrder 会员购买订单（对齐 membership_orders）
type MembershipOrder struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	OrderNo         string     `gorm:"size:64;uniqueIndex" json:"order_no"`
	UserID          uint       `json:"user_id"`
	MembershipLevel string     `gorm:"size:50" json:"membership_level"`
	Amount          float64    `gorm:"type:decimal(10,2)" json:"amount"`
	Days            int        `json:"days"`
	Status          string     `gorm:"size:20;default:paid" json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	PaidAt          *time.Time `json:"paid_at"`
}

func (MembershipOrder) TableName() string { return "membership_orders" }

// PaymentOrder 充值订单（对齐 payment_orders）
type PaymentOrder struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	OrderNo       string     `gorm:"size:64;uniqueIndex" json:"order_no"`
	UserID        uint       `json:"user_id"`
	Amount        float64    `gorm:"type:decimal(10,2)" json:"amount"`
	Points        int        `gorm:"default:0" json:"points"` // 实际为赠送经验
	Status        string     `gorm:"size:20;default:pending" json:"status"`
	PaymentMethod string     `gorm:"size:20;default:''" json:"payment_method"`
	PaymentScene  string     `gorm:"size:20;default:''" json:"payment_scene"`
	TradeNo       string     `gorm:"size:64;default:''" json:"trade_no"`
	OpenID        string     `gorm:"size:128;default:''" json:"openid"`
	CreatedAt     time.Time  `json:"created_at"`
	PaidAt        *time.Time `json:"paid_at"`
}

func (PaymentOrder) TableName() string { return "payment_orders" }

// LinkPurchase 链接购买记录（对齐 link_purchases）
type LinkPurchase struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"uniqueIndex:unique_purchase" json:"user_id"`
	KeywordID    uint      `gorm:"uniqueIndex:unique_purchase" json:"keyword_id"`
	BlockIndex   int       `gorm:"uniqueIndex:unique_purchase" json:"block_index"`
	BalanceSpent float64   `gorm:"type:decimal(10,2);default:0.00" json:"balance_spent"`
	CreatedAt    time.Time `json:"created_at"`
}

func (LinkPurchase) TableName() string { return "link_purchases" }

// KeywordFavorite 关键词收藏记录（user_id + keyword_id 复合唯一索引防重复）
type KeywordFavorite struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:unique_favorite" json:"user_id"`
	KeywordID uint      `gorm:"uniqueIndex:unique_favorite" json:"keyword_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (KeywordFavorite) TableName() string { return "keyword_favorites" }

// PointsTransaction 余额/经验流水（对齐 points_transactions）
type PointsTransaction struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `json:"user_id"`
	Type          string    `gorm:"size:20" json:"type"`
	Amount        float64   `gorm:"type:decimal(10,2);default:0.00" json:"amount"`
	BalanceAmount float64   `gorm:"type:decimal(10,2);default:0.00" json:"balance_amount"`
	ExpEarned     int       `gorm:"default:0" json:"exp_earned"`
	Description   string    `gorm:"size:255;default:''" json:"description"`
	RefID         *uint     `json:"ref_id"`
	CreatedAt     time.Time `json:"created_at"`
}

func (PointsTransaction) TableName() string { return "points_transactions" }

// MembershipReminder 会员到期提醒记录（对齐 membership_reminders）
type MembershipReminder struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserID          uint      `gorm:"index:idx_user_days" json:"user_id"`
	MembershipLevel string    `gorm:"size:50" json:"membership_level"`
	DaysLeft        int       `gorm:"index:idx_user_days" json:"days_left"`
	SentAt          time.Time `json:"sent_at"`
}

func (MembershipReminder) TableName() string { return "membership_reminders" }

// EmailVerification 邮箱验证码（对齐 email_verification）
type EmailVerification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Email     string    `gorm:"size:100;index:idx_email_purpose" json:"email"`
	Code      string    `gorm:"size:10" json:"code"`
	Purpose   string    `gorm:"size:30;index:idx_email_purpose" json:"purpose"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `gorm:"default:false" json:"used"`
	CreatedAt time.Time `json:"created_at"`
}

func (EmailVerification) TableName() string { return "email_verification" }

// SchemaMigration 迁移版本跟踪（新增，对齐 schema_migrations）
type SchemaMigration struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Version   string    `gorm:"size:255;uniqueIndex" json:"version"`
	AppliedAt time.Time `json:"applied_at"`
}

func (SchemaMigration) TableName() string { return "schema_migrations" }

// MediaFile 媒体库文件元数据（对齐 media_files）
type MediaFile struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	URL        string    `gorm:"size:255;index" json:"url"` // /uploads/media/<filename>
	Filename   string    `gorm:"size:255" json:"filename"`  // 物理文件名
	OriginName string    `gorm:"size:255;default:''" json:"origin_name"`
	Mime       string    `gorm:"size:100;default:''" json:"mime"`
	Size       int64     `gorm:"default:0" json:"size"` // 字节
	Width      int       `gorm:"default:0" json:"width"`
	Height     int       `gorm:"default:0" json:"height"`
	Source     string    `gorm:"size:20;default:local" json:"source"` // local | remote
	UploaderID uint      `gorm:"default:0" json:"uploader_id"`
	CreatedAt  time.Time `json:"created_at"`
}

func (MediaFile) TableName() string { return "media_files" }

// AllModels 返回所有需要 AutoMigrate 的模型
func AllModels() []interface{} {
	return []interface{}{
		&User{},
		&KeywordResponse{},
		&KeywordCategory{},
		&KeywordCategoryLink{},
		&Page{},
		&Announcement{},
		&SiteSetting{},
		&CheckInConfig{},
		&CheckInRecord{},
		&LevelConfig{},
		&MembershipConfig{},
		&MembershipOrder{},
		&PaymentOrder{},
		&LinkPurchase{},
		&KeywordFavorite{},
		&PointsTransaction{},
		&MembershipReminder{},
		&EmailVerification{},
		&MediaFile{},
		&SchemaMigration{},
	}
}

// Migrate 执行 AutoMigrate
func Migrate(db *gorm.DB) error {
	// 在迁移前先清理孤立数据，避免外键约束失败
	if err := cleanOrphanedData(db); err != nil {
		return err
	}
	return db.AutoMigrate(AllModels()...)
}

// cleanOrphanedData 清理孤立的关联数据
func cleanOrphanedData(db *gorm.DB) error {
	if !db.Migrator().HasTable("keyword_category_links") {
		return nil
	}
	// 清理 keyword_category_links 中引用不存在的 keyword_id 的记录
	sql := `DELETE FROM keyword_category_links
	        WHERE keyword_id NOT IN (SELECT id FROM keyword_responses)`
	if err := db.Exec(sql).Error; err != nil {
		// 如果表不存在，忽略错误（首次安装时）
		if db.Migrator().HasTable("keyword_category_links") {
			return err
		}
	}

	// 清理 keyword_category_links 中引用不存在的 category_id 的记录
	sql = `DELETE FROM keyword_category_links
	       WHERE category_id NOT IN (SELECT id FROM keyword_categories)`
	if err := db.Exec(sql).Error; err != nil {
		if db.Migrator().HasTable("keyword_category_links") {
			return err
		}
	}

	return nil
}
