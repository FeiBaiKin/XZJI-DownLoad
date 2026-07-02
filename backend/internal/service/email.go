package service

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/smtp"

	"gorm.io/gorm"
	"xzji/backend/internal/models"
)

// EmailConfig 从 site_settings 表读取的邮件配置
type EmailConfig struct {
	SMTPHost           string
	SMTPPort           int
	SMTPUser           string
	SMTPPass           string
	FromName           string
	ForceVerify        bool
	EnablePasswordless bool
}

// GetEmailConfig 从 site_settings 读取邮件配置
func GetEmailConfig(db *gorm.DB) (EmailConfig, error) {
	var rows []models.SiteSetting
	if err := db.Where("setting_key LIKE ?", "email_%").Find(&rows).Error; err != nil {
		return EmailConfig{}, err
	}
	cfg := EmailConfig{SMTPPort: 465}
	for _, r := range rows {
		switch r.SettingKey {
		case "email_smtp_host":
			cfg.SMTPHost = r.SettingValue
		case "email_smtp_port":
			fmt.Sscanf(r.SettingValue, "%d", &cfg.SMTPPort)
		case "email_smtp_user":
			cfg.SMTPUser = r.SettingValue
		case "email_smtp_pass":
			cfg.SMTPPass = r.SettingValue
		case "email_from_name":
			cfg.FromName = r.SettingValue
		case "email_force_verify":
			cfg.ForceVerify = r.SettingValue == "1"
		case "email_enable_passwordless":
			cfg.EnablePasswordless = r.SettingValue == "1"
		}
	}
	return cfg, nil
}

// GenerateCode 生成 6 位随机数字验证码（对齐原 generateCode）
func GenerateCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "123456" // 兜底不应发生
	}
	return fmt.Sprintf("%06d", n.Int64()+100000)
}

// SendVerificationCode 发送验证码邮件（HTML 格式，对齐 Node）
func SendVerificationCode(cfg EmailConfig, to, code, purpose string) error {
	purposeLabel := map[string]string{
		"register":        "注册账号",
		"forgot-password": "找回密码",
		"login":           "邮箱登录",
	}[purpose]
	if purposeLabel == "" {
		purposeLabel = purpose
	}
	subject := fmt.Sprintf("%s - %s验证码", cfg.FromName, purposeLabel)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <style>
    body { margin:0; padding:0; background:#f5f5f5; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; }
    .container { max-width:480px; margin:40px auto; background:#ffffff; border:1px solid #e5e5e5; }
    .header { padding:28px 32px 0; }
    .header h2 { margin:0; font-size:18px; font-weight:600; color:#0a0a0a; }
    .body { padding:20px 32px 32px; }
    .code-box { background:#f9f9f9; border:1px solid #e5e5e5; padding:24px; text-align:center; margin:24px 0; }
    .code { font-family: 'JetBrains Mono', 'Courier New', monospace; font-size:32px; font-weight:700; letter-spacing:8px; color:#0a0a0a; }
    .meta { font-size:13px; color:#888; line-height:1.6; }
    .footer { padding:16px 32px; border-top:1px solid #f0f0f0; font-size:12px; color:#aaa; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h2>%s · %s</h2>
    </div>
    <div class="body">
      <p class="meta">您正在使用 <strong>%s</strong> 功能，验证码如下：</p>
      <div class="code-box">
        <span class="code">%s</span>
      </div>
      <p class="meta">验证码 10 分钟内有效。如非本人操作，请忽略此邮件。</p>
    </div>
    <div class="footer">
      此邮件由系统自动发送，请勿回复
    </div>
  </div>
</body>
</html>`, cfg.FromName, purposeLabel, purposeLabel, code)

	return sendMail(cfg, to, subject, html, "text/html")
}

// SendHTMLMail 发送 HTML 邮件（用于会员到期提醒等）
func SendHTMLMail(cfg EmailConfig, to, subject, html string) error {
	return sendMail(cfg, to, subject, html, "text/html")
}

// sendMail 统一邮件发送入口：构造 MIME 报文并按端口选择 SSL 直连（465）或 STARTTLS（587 等）。
func sendMail(cfg EmailConfig, to, subject, body, contentType string) error {
	if cfg.SMTPHost == "" || cfg.SMTPUser == "" {
		return fmt.Errorf("SMTP 未配置")
	}
	headers := fmt.Sprintf("From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: %s; charset=UTF-8\r\n\r\n",
		cfg.FromName, cfg.SMTPUser, to, encodeHeader(subject), contentType)
	msg := []byte(headers + body)

	var auth smtp.Auth
	if cfg.SMTPPass != "" {
		if cfg.SMTPPort == 465 {
			// 465 为隐式 TLS，连接已加密，需用不校验 server.TLS 的自定义认证器
			auth = &plainAuth{username: cfg.SMTPUser, password: cfg.SMTPPass}
		} else {
			auth = smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)
		}
	}
	if cfg.SMTPPort == 465 {
		return sendMailSSL(cfg.SMTPHost, cfg.SMTPPort, auth, cfg.SMTPUser, []string{to}, msg)
	}
	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)
	return smtp.SendMail(addr, auth, cfg.SMTPUser, []string{to}, msg)
}

// plainAuth 实现 SASL PLAIN 认证，但不做 net/smtp PlainAuth 的「未加密连接即拒绝」校验。
// 用于 465 隐式 TLS：底层已是 TLS 加密连接，但 smtp.NewClient 不会把 server.TLS 置真，
// 标准库 PlainAuth 会误报 ErrUnencrypted，故此处自行实现。
type plainAuth struct{ username, password string }

func (a *plainAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "PLAIN", []byte("\x00" + a.username + "\x00" + a.password), nil
}

func (a *plainAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		return nil, fmt.Errorf("unexpected server challenge")
	}
	return nil, nil
}

// sendMailSSL SMTP 隐式 TLS 直连（端口 465）。
// net/smtp 未原生支持 465，需先用 tls.Dial 建立 TLS 连接，再在其上跑 SMTP 会话。
func sendMailSSL(host string, port int, auth smtp.Auth, from string, to []string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("SMTP SSL 连接失败: %w", err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("SMTP 客户端创建失败: %w", err)
	}
	defer c.Quit()

	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("SMTP 认证失败: %w", err)
		}
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := c.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	return w.Close()
}

// encodeHeader 将含非 ASCII（如中文）的邮件头字段编码为 RFC 2047 B 编码，
// 避免裸 UTF-8 头被部分 MTA 拒收。
func encodeHeader(s string) string {
	return "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(s)) + "?="
}
