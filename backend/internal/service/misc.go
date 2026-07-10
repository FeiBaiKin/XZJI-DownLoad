package service

import (
	"errors"
	"fmt"
	"time"

	gomysql "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"xzji/backend/internal/models"
)

// ====== 签到 ======

func CheckInStatus(db *gorm.DB, userID uint) (map[string]any, error) {
	// 签到日界线统一用服务器本地时区（与写入侧、DashboardStats 一致），不可用 UTC：
	// DSN loc=Local 下时间参数按本地时区序列化，混用 UTC 会在本地 0 点到 UTC 0 点
	// 的时间窗内查错一天
	now := time.Now()
	today := now.Format("2006-01-02")
	var todayRec models.CheckInRecord
	// 用 Find 替代 First：用户今日未签到是正常情况，不应触发 GORM "record not found" 日志
	db.Where("user_id = ? AND check_in_date = ?", userID, today).Limit(1).Find(&todayRec)
	checkedIn := todayRec.ID > 0

	// 优化：一次性获取所有签到记录，在内存中计算连续天数（避免 N+1 查询）
	var records []models.CheckInRecord
	db.Where("user_id = ?", userID).Order("check_in_date DESC").Select("check_in_date").Find(&records)

	// 构建日期集合用于快速查找。DATE 列被驱动解析为 DSN loc 时区的当日零点，
	// 直接 Format 即还原存储日期；再做 .UTC() 会在东八区把日期回退一天
	dateSet := make(map[string]bool)
	for _, r := range records {
		dateSet[r.CheckInDate.Format("2006-01-02")] = true
	}

	// 计算连续签到天数
	streak := 0
	cursor := now
	if !checkedIn {
		cursor = cursor.AddDate(0, 0, -1)
	}
	for {
		ds := cursor.Format("2006-01-02")
		if !dateSet[ds] {
			break
		}
		streak++
		cursor = cursor.AddDate(0, 0, -1)
	}

	var cfg models.CheckInConfig
	db.First(&cfg)
	if cfg.ConsecutiveDays == 0 { cfg.ConsecutiveDays = 7 }
	if cfg.DailyBalance == 0 { cfg.DailyBalance = 0.50 }
	if cfg.ConsecutiveBonusBalance == 0 { cfg.ConsecutiveBonusBalance = 2.00 }
	if cfg.DailyExp == 0 { cfg.DailyExp = 10 }

	tomorrowStreak := streak + 1
	tomorrowBonus := tomorrowStreak >= cfg.ConsecutiveDays
	tomorrowBalance := cfg.DailyBalance
	if tomorrowBonus { tomorrowBalance += cfg.ConsecutiveBonusBalance }

	return map[string]any{
		"checkedIn": checkedIn, "streak": streak,
		"todayBalance": todayRec.BalanceEarned, "todayExp": todayRec.ExpEarned, "isBonus": todayRec.IsBonus,
		"config": map[string]any{
			"dailyBalance": cfg.DailyBalance, "consecutiveDays": cfg.ConsecutiveDays,
			"consecutiveBonusBalance": cfg.ConsecutiveBonusBalance, "dailyExp": cfg.DailyExp,
		},
		"tomorrowBalance": tomorrowBalance,
	}, nil
}

func CheckIn(db *gorm.DB, userID uint) (map[string]any, error) {
	// 日界线与 CheckInStatus、写入的 check_in_date 保持同一本地时区，见 CheckInStatus 注释
	now := time.Now()
	today := now.Format("2006-01-02")

	// 事务外读取签到配置（只读，无需加锁）
	var cfg models.CheckInConfig
	db.First(&cfg)
	if cfg.DailyBalance == 0 { cfg.DailyBalance = 0.50 }
	if cfg.ConsecutiveDays == 0 { cfg.ConsecutiveDays = 7 }
	if cfg.ConsecutiveBonusBalance == 0 { cfg.ConsecutiveBonusBalance = 2.00 }
	if cfg.DailyExp == 0 { cfg.DailyExp = 10 }

	// 优化：一次性获取所有签到记录，在内存中计算连续天数（避免 N+1 查询）
	var records []models.CheckInRecord
	db.Where("user_id = ?", userID).Order("check_in_date DESC").Select("check_in_date").Find(&records)
	
	// 构建日期集合用于快速查找（DATE 列直接 Format 还原存储日期，禁用 .UTC()，见 CheckInStatus 注释）
	dateSet := make(map[string]bool)
	for _, r := range records {
		dateSet[r.CheckInDate.Format("2006-01-02")] = true
	}

	// 计算连续签到天数（从昨天开始，因为今天还没签到）
	streak := 0
	cursor := now.AddDate(0, 0, -1)
	for {
		ds := cursor.Format("2006-01-02")
		if !dateSet[ds] {
			break
		}
		streak++
		cursor = cursor.AddDate(0, 0, -1)
	}
	newStreak := streak + 1
	bal := cfg.DailyBalance
	exp := cfg.DailyExp
	isBonus := false
	if newStreak >= cfg.ConsecutiveDays {
		bal += cfg.ConsecutiveBonusBalance
		exp += exp / 2
		isBonus = true
	}

	// 事务内执行写操作：检查重复签到 + 创建记录 + 更新余额 + 创建流水
	var user models.User
	err := db.Transaction(func(tx *gorm.DB) error {
		// 锁定用户行，防止并发签到（GORM v2 中 gorm:query_option 已失效，必须用 clause.Locking）
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			return fmt.Errorf("用户不存在")
		}

		// 检查今日是否已签到（Find 避免未签到时的 "record not found" 噪音日志）
		var exist models.CheckInRecord
		tx.Where("user_id = ? AND check_in_date = ?", userID, today).Limit(1).Find(&exist)
		if exist.ID > 0 {
			return fmt.Errorf("今日已签到，请明天再来")
		}

		// 创建签到记录：写入本地日期零点，保证与 today 查询条件、unique_daily 落在同一天
		if err := tx.Create(&models.CheckInRecord{
			UserID:        userID,
			CheckInDate:   time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()),
			BalanceEarned: bal, ExpEarned: exp, IsBonus: isBonus,
		}).Error; err != nil {
			// 撞 unique_daily 唯一键说明并发下已有当日记录，按已签到返回而非暴露内部错误
			var mysqlErr *gomysql.MySQLError
			if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
				return fmt.Errorf("今日已签到，请明天再来")
			}
			return fmt.Errorf("创建签到记录失败: %w", err)
		}

		// 更新用户余额和经验
		if err := tx.Model(&user).Updates(map[string]any{
			"balance": gorm.Expr("balance + ?", bal),
			"exp":     gorm.Expr("exp + ?", exp),
		}).Error; err != nil {
			return fmt.Errorf("更新用户余额失败: %w", err)
		}

		// 创建积分流水
		desc := "每日签到奖励"
		txType := "checkin"
		if isBonus {
			txType = "bonus"
			desc = fmt.Sprintf("连续签到%d天奖励", newStreak)
		}
		if err := tx.Create(&models.PointsTransaction{
			UserID: userID, Type: txType, Amount: bal,
			ExpEarned: exp, Description: desc,
		}).Error; err != nil {
			return fmt.Errorf("创建积分流水失败: %w", err)
		}

		// 重新加载用户数据以返回最新值
		return tx.First(&user, userID).Error
	})

	if err != nil {
		return nil, err
	}

	// 事务外执行等级升级检查（独立操作，失败不影响签到）
	lvlResult := CheckLevelUp(db, userID)

	return map[string]any{
		"message": "签到成功", "balanceEarned": bal, "expEarned": exp,
		"totalBalance": user.Balance, "totalExp": user.Exp, "level": user.Level,
		"streak": newStreak, "isBonus": isBonus,
		"leveledUp": lvlResult["leveledUp"], "newLevel": lvlResult["newLevel"], "levelLabel": lvlResult["label"],
	}, nil
}

// ====== 等级 ======

func CheckLevelUp(db *gorm.DB, userID uint) map[string]any {
	var user models.User; db.First(&user, userID)
	var lvl models.LevelConfig
	err := db.Where("exp_required <= ?", user.Exp).Order("level DESC").First(&lvl).Error
	if err == nil && lvl.Level > user.Level {
		db.Model(&user).Update("level", lvl.Level)
		return map[string]any{"leveledUp": true, "newLevel": lvl.Level, "label": lvl.Label}
	}
	return map[string]any{"leveledUp": false}
}

func LevelInfo(db *gorm.DB, userID uint) (map[string]any, error) {
	CheckLevelUp(db, userID)
	var user models.User; db.First(&user, userID)
	var currentCfg models.LevelConfig
	db.Where("level = ?", user.Level).First(&currentCfg)
	var nextCfg models.LevelConfig
	db.Where("level = ?", user.Level+1).First(&nextCfg)
	expInLevel := user.Exp - currentCfg.ExpRequired
	expNeeded := 0
	progress := 100
	if nextCfg.ID > 0 {
		expNeeded = nextCfg.ExpRequired - currentCfg.ExpRequired
		if expNeeded > 0 {
			progress = expInLevel * 100 / expNeeded
			if progress > 100 { progress = 100 }
		}
	}
	var allLevels []models.LevelConfig
	db.Order("level ASC").Find(&allLevels)
	return map[string]any{
		"level": user.Level, "label": currentCfg.Label, "exp": user.Exp,
		"expInCurrentLevel": expInLevel, "expNeeded": expNeeded, "progress": progress,
		"nextLevel": nextCfg.Level, "nextLabel": nextCfg.Label, "allLevels": allLevels,
	}, nil
}

// ====== 仪表盘统计 ======

func DashboardStats(db *gorm.DB) (map[string]any, error) {
	today := time.Now().Format("2006-01-02")
	weekAgo := time.Now().AddDate(0, 0, -7)
	monthAgo := time.Now().AddDate(0, 0, -30)

	// 优化：关键词数量（单独查询，无法合并）
	var kwCnt int64
	db.Model(&models.KeywordResponse{}).Count(&kwCnt)

	// 优化：合并用户统计查询（总数 + 今日/本周/本月新增）
	type userStats struct {
		Total     int64
		NewToday  int64
		NewWeek   int64
		NewMonth  int64
		ActiveMem int64
		ExpMem    int64
	}
	var us userStats
	db.Raw(`
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN DATE(created_at) = ? THEN 1 ELSE 0 END) as new_today,
			SUM(CASE WHEN created_at >= ? THEN 1 ELSE 0 END) as new_week,
			SUM(CASE WHEN created_at >= ? THEN 1 ELSE 0 END) as new_month,
			SUM(CASE WHEN membership_level IS NOT NULL AND membership_expires_at > NOW() THEN 1 ELSE 0 END) as active_mem,
			SUM(CASE WHEN membership_level IS NOT NULL AND membership_expires_at > NOW() AND membership_expires_at <= DATE_ADD(NOW(), INTERVAL 7 DAY) THEN 1 ELSE 0 END) as exp_mem
		FROM users
	`, today, weekAgo, monthAgo).Scan(&us)

	// 优化：合并订单统计查询（总数/已支付/待支付 + 收入统计）
	type orderStats struct {
		Total      int64
		Paid       int64
		Pending    int64
		RevTotal   float64
		RevMonth   float64
		RevToday   float64
	}
	var os orderStats
	db.Raw(`
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN status = 'paid' THEN 1 ELSE 0 END) as paid,
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
			COALESCE(SUM(CASE WHEN status = 'paid' THEN amount ELSE 0 END), 0) as rev_total,
			COALESCE(SUM(CASE WHEN status = 'paid' AND paid_at >= ? THEN amount ELSE 0 END), 0) as rev_month,
			COALESCE(SUM(CASE WHEN status = 'paid' AND DATE(paid_at) = ? THEN amount ELSE 0 END), 0) as rev_today
		FROM payment_orders
	`, monthAgo, today).Scan(&os)

	// 今日签到数（单独查询）
	var ckToday int64
	db.Model(&models.CheckInRecord{}).Where("check_in_date = ?", today).Count(&ckToday)

	// 站点名称（单独查询）
	var siteName string
	db.Raw("SELECT setting_value FROM site_settings WHERE setting_key = 'siteName' LIMIT 1").Scan(&siteName)

	// 支付方式分布（已优化，使用 GROUP BY）
	type methodStat struct {
		PaymentMethod string  `json:"payment_method"`
		Count         int64   `json:"count"`
		Revenue       float64 `json:"revenue"`
	}
	var methods []methodStat
	db.Raw(`SELECT payment_method, COUNT(*) as count, COALESCE(SUM(amount),0) as revenue
		FROM payment_orders WHERE status = 'paid' GROUP BY payment_method ORDER BY revenue DESC`).Scan(&methods)

	return map[string]any{
		"keywords": kwCnt, "users": us.Total, "newUsersToday": us.NewToday, "newUsersWeek": us.NewWeek, "newUsersMonth": us.NewMonth,
		"revenueTotal": os.RevTotal, "revenueMonth": os.RevMonth, "revenueToday": os.RevToday,
		"ordersTotal": os.Total, "ordersPaid": os.Paid, "ordersPending": os.Pending,
		"activeMemberships": us.ActiveMem, "expiringMemberships": us.ExpMem, "checkinsToday": ckToday,
		"siteName": siteName, "methodBreakdown": methods,
	}, nil
}

// ====== 站点设置 ======

func SiteSettings(db *gorm.DB) (map[string]string, error) {
	var rows []models.SiteSetting
	db.Find(&rows)
	m := map[string]string{}
	for _, r := range rows { m[r.SettingKey] = r.SettingValue }
	if _, ok := m["siteName"]; !ok { m["siteName"] = "XZJI" }
	return m, nil
}

func UpdateSiteSettings(db *gorm.DB, settings map[string]string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for k, v := range settings {
			if err := tx.Exec("INSERT INTO site_settings (setting_key, setting_value) VALUES (?, ?) ON DUPLICATE KEY UPDATE setting_value = ?", k, v, v).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ====== 邮件配置 ======

func UpdateEmailConfig(db *gorm.DB, settings map[string]string) error {
	return UpdateSiteSettings(db, settings)
}

func EmailTest(db *gorm.DB, to string) error {
	cfg, err := GetEmailConfig(db)
	if err != nil {
		return err
	}
	if cfg.SMTPHost == "" {
		return fmt.Errorf("SMTP 未配置")
	}

	subject := fmt.Sprintf("%s - 测试邮件", cfg.FromName)
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <style>
    body { margin:0; padding:0; background:#f5f5f5; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; }
    .container { max-width:480px; margin:40px auto; background:#ffffff; border:1px solid #e5e5e5; }
    .header { padding:28px 32px 0; }
    .header h2 { margin:0; font-size:18px; font-weight:600; color:#0a0a0a; }
    .body { padding:20px 32px 32px; line-height:1.7; color:#333; }
    .footer { padding:16px 32px; border-top:1px solid #f0f0f0; font-size:12px; color:#aaa; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h2>%s - 测试邮件</h2>
    </div>
    <div class="body">
      <p>这是一封测试邮件，表明 SMTP 配置正确。</p>
      <p>发送时间：%s</p>
    </div>
    <div class="footer">
      此邮件由系统自动发送，请勿回复
    </div>
  </div>
</body>
</html>`, cfg.FromName, time.Now().Format("2006-01-02 15:04:05"))

	return SendHTMLMail(cfg, to, subject, html)
}
