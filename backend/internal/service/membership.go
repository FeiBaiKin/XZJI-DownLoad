package service

import (
	"fmt"
	"log/slog"
	"math"
	"time"

	"gorm.io/gorm"
	"xzji/backend/internal/models"
)

// CheckMembershipExpiryReminders 扫描 7 天内到期的会员，按 7/3/1 天阈值发送提醒邮件。
// 对齐原项目 index.js::checkMembershipExpiryReminders：
//   - 仅对 membership_level/membership_expires_at 非空且有邮箱的用户
//   - daysLeft = ceil((expires - now) / 1d)，仅在阈值 {7,3,1} 时提醒
//   - 同一 (user, days_left) 仅提醒一次（membership_reminders 去重）
//   - 发送成功后写入 membership_reminders
//
// 返回本次发送的提醒数量。
func CheckMembershipExpiryReminders(db *gorm.DB) (int, error) {
	cfg, err := GetEmailConfig(db)
	if err != nil {
		return 0, err
	}
	// SMTP 未配置则跳过（与原项目一致）
	if cfg.SMTPHost == "" || cfg.SMTPUser == "" {
		slog.Info("会员到期提醒: SMTP 未配置，跳过")
		return 0, nil
	}

	// 查询 7 天内到期的会员
	type expiringUser struct {
		ID                  uint      `gorm:"column:id"`
		Username            string    `gorm:"column:username"`
		Email               string    `gorm:"column:email"`
		MembershipLevel     string    `gorm:"column:membership_level"`
		MembershipExpiresAt time.Time `gorm:"column:membership_expires_at"`
	}
	var users []expiringUser
	if err := db.Raw(`SELECT id, username, email, membership_level, membership_expires_at
		FROM users
		WHERE membership_level IS NOT NULL
		  AND membership_expires_at IS NOT NULL
		  AND email != ''
		  AND membership_expires_at > NOW()
		  AND membership_expires_at <= NOW() + INTERVAL '7 days'`).Scan(&users).Error; err != nil {
		return 0, err
	}
	if len(users) == 0 {
		return 0, nil
	}

	thresholds := map[int]bool{7: true, 3: true, 1: true}
	sent := 0
	for _, u := range users {
		daysLeft := int(math.Ceil(u.MembershipExpiresAt.Sub(time.Now()).Hours() / 24))
		if !thresholds[daysLeft] {
			continue
		}
		// 同一天数已提醒过则跳过
		var cnt int64
		db.Model(&models.MembershipReminder{}).Where("user_id = ? AND days_left = ?", u.ID, daysLeft).Count(&cnt)
		if cnt > 0 {
			continue
		}

		label := fmt.Sprintf("%d 天后", daysLeft)
		if daysLeft == 1 {
			label = "明天"
		}
		dateStr := u.MembershipExpiresAt.Format("2006/01/02")
		subject := fmt.Sprintf("%s - 会员到期提醒", cfg.FromName)
		html := buildReminderHTML(cfg.FromName, u.Username, u.MembershipLevel, label, dateStr)

		if err := SendHTMLMail(cfg, u.Email, subject, html); err != nil {
			slog.Error("会员提醒发送失败", "username", u.Username, "err", err)
			continue
		}
		db.Create(&models.MembershipReminder{
			UserID:          u.ID,
			MembershipLevel: u.MembershipLevel,
			DaysLeft:        daysLeft,
			SentAt:          time.Now(),
		})
		sent++
		slog.Info("会员到期提醒已发送", "username", u.Username, "level", u.MembershipLevel, "daysLeft", daysLeft)
	}

	if sent > 0 {
		slog.Info("会员到期提醒完成", "sent", sent)
	}
	return sent, nil
}

// buildReminderHTML 构造会员到期提醒邮件正文（对齐原项目 HTML 模板）
func buildReminderHTML(fromName, username, level, label, dateStr string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head><meta charset="UTF-8"><style>
  body { margin:0; padding:0; background:#f5f5f5; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; }
  .container { max-width:480px; margin:40px auto; background:#fff; border:1px solid #e5e5e5; }
  .header { padding:28px 32px 0; }
  .header h2 { margin:0; font-size:18px; font-weight:600; color:#0a0a0a; }
  .body { padding:20px 32px 32px; line-height:1.7; color:#333; }
  .highlight { font-weight:700; color:#0a0a0a; }
  .footer { padding:16px 32px; border-top:1px solid #f0f0f0; font-size:12px; color:#aaa; }
</style></head>
<body>
  <div class="container">
    <div class="header"><h2>%s - 会员到期提醒</h2></div>
    <div class="body">
      <p>尊敬的 <span class="highlight">%s</span>，你好！</p>
      <p>你的 <span class="highlight">%s</span> 会员将在 <span class="highlight">%s</span>（%s）到期。</p>
      <p>到期后将无法享受会员专属权益，请及时续费。</p>
    </div>
    <div class="footer">此邮件由系统自动发送，请勿回复</div>
  </div>
</body></html>`, fromName, username, level, label, dateStr)
}
