package service

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"log/slog"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"xzji/backend/internal/models"
	"xzji/backend/internal/util"
)

// usernameRegex 用户名仅允许英文字母、数字和下划线（不支持中文等字符）
var usernameRegex = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// AuthService 认证相关业务逻辑
type AuthService struct {
	DB        *gorm.DB
	JWTSecret string
}

// ====== 登录 ======

// LoginRequest 用户名+密码登录
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResult 登录成功返回
type LoginResult struct {
	Message string    `json:"message"`
	Token   string    `json:"token"`
	User    UserBrief `json:"user"`
}

// UserBrief 返回给前端的用户摘要
type UserBrief struct {
	ID                  uint       `json:"id"`
	Username            string     `json:"username"`
	Role                string     `json:"role"`
	Email               string     `json:"email"`
	Nickname            string     `json:"nickname"`
	Avatar              string     `json:"avatar"`
	Balance             float64    `json:"balance"`
	Exp                 int        `json:"exp"`
	Level               int        `json:"level"`
	LevelLabel          string     `json:"level_label"` // 后台 level_config 配置的等级名称，未配置时为空串
	MembershipLevel     *string    `json:"membership_level"`
	MembershipExpiresAt *time.Time `json:"membership_expires_at"`
	CreatedAt           time.Time  `json:"created_at"`
}

// toUserBrief 转换模型为 brief。
// 需要 db 是为了带出后台配置的等级名称（level_config.label），
// 前端徽章优先显示它，缺配置时才回退 "Lv.N"。
func toUserBrief(db *gorm.DB, u *models.User) UserBrief {
	b := UserBrief{
		ID:                  u.ID,
		Username:            u.Username,
		Role:                u.Role,
		Email:               u.Email,
		Nickname:            u.Nickname,
		Avatar:              u.Avatar,
		Balance:             u.Balance,
		Exp:                 u.Exp,
		Level:               u.Level,
		LevelLabel:          levelLabelOf(db, u.Level),
		MembershipLevel:     u.MembershipLevel,
		MembershipExpiresAt: u.MembershipExpiresAt,
		CreatedAt:           u.CreatedAt,
	}
	if b.Nickname == "" {
		b.Nickname = u.Username
	}
	if b.Avatar == "" {
		b.Avatar = ""
	}
	return b
}

// levelLabelOf 查后台配置的等级名称；未配置该等级时返回空串（由前端回退展示）
func levelLabelOf(db *gorm.DB, level int) string {
	if db == nil || level <= 0 {
		return ""
	}
	var cfg models.LevelConfig
	if err := db.Select("label").Where("level = ?", level).First(&cfg).Error; err != nil {
		return ""
	}
	return cfg.Label
}

// Login 用户名密码登录
func (s *AuthService) Login(req LoginRequest) (*LoginResult, error) {
	var user models.User
	if err := s.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		return nil, fmt.Errorf("用户名或密码错误")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("用户名或密码错误")
	}
	token, err := s.generateToken(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, fmt.Errorf("令牌生成失败")
	}
	return &LoginResult{
		Message: "登录成功",
		Token:   token,
		User:    toUserBrief(s.DB, &user),
	}, nil
}

// ====== 注册 ======

// RegisterRequest 注册
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Code     string `json:"code"`
}

// Register 注册（含邮箱可选/强制验证逻辑）
func (s *AuthService) Register(req RegisterRequest) (*LoginResult, error) {
	req.Username = strings.TrimSpace(req.Username)
	if len(req.Username) < 3 {
		return nil, fmt.Errorf("用户名至少需要3个字符")
	}
	if len(req.Username) > 50 {
		return nil, fmt.Errorf("用户名最多50个字符")
	}
	if !usernameRegex.MatchString(req.Username) {
		return nil, fmt.Errorf("用户名仅支持英文字母、数字和下划线，不支持中文")
	}
	if len(req.Password) < 6 {
		return nil, fmt.Errorf("密码至少需要6个字符")
	}
	// 检查用户名唯一
	var exist int64
	s.DB.Model(&models.User{}).Where("username = ?", req.Username).Count(&exist)
	if exist > 0 {
		return nil, fmt.Errorf("用户名已存在")
	}

	emailCfg, _ := GetEmailConfig(s.DB)
	normEmail := strings.TrimSpace(strings.ToLower(req.Email))

	if emailCfg.ForceVerify {
		// 强制验证模式：邮箱+验证码必填
		if normEmail == "" {
			return nil, fmt.Errorf("请提供邮箱地址")
		}
		if req.Code == "" {
			return nil, fmt.Errorf("请提供邮箱验证码")
		}
		var emailCnt int64
		s.DB.Model(&models.User{}).Where("email = ?", normEmail).Count(&emailCnt)
		if emailCnt > 0 {
			return nil, fmt.Errorf("邮箱已被注册")
		}
		// 验证验证码
		if err := consumeCode(s.DB, normEmail, req.Code, "register"); err != nil {
			return nil, err
		}
		return s.createUser(req.Username, req.Password, normEmail, true)
	}

	// 非强制验证：邮箱可选
	if normEmail != "" {
		var emailCnt int64
		s.DB.Model(&models.User{}).Where("email = ?", normEmail).Count(&emailCnt)
		if emailCnt > 0 {
			return nil, fmt.Errorf("邮箱已被注册")
		}
	}
	return s.createUser(req.Username, req.Password, normEmail, false)
}

func (s *AuthService) createUser(username, password, email string, emailVerified bool) (*LoginResult, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败")
	}
	user := models.User{
		Username:      username,
		Password:      string(hashed),
		Role:          "user",
		Email:         email,
		Nickname:      username,
		Balance:       0,
		Exp:           0,
		Level:         1,
		EmailVerified: emailVerified,
	}
	if err := s.DB.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("创建用户失败")
	}
	token, err := s.generateToken(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, fmt.Errorf("令牌生成失败")
	}
	return &LoginResult{
		Message: "注册成功",
		Token:   token,
		User:    toUserBrief(s.DB, &user),
	}, nil
}

// ====== 邮箱验证码 ======

// SendCodeRequest 发送验证码
type SendCodeRequest struct {
	Email   string `json:"email"`
	Purpose string `json:"purpose"`
}

// SendCode 发送验证码（含频率限制、邮箱校验）
func (s *AuthService) SendCode(req SendCodeRequest) error {
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		return fmt.Errorf("邮箱格式不正确")
	}
	validPurposes := map[string]bool{"register": true, "forgot-password": true, "login": true}
	if !validPurposes[req.Purpose] {
		return fmt.Errorf("无效的验证码用途")
	}

	emailCfg, _ := GetEmailConfig(s.DB)

	// 用途特定检查
	if req.Purpose == "register" && emailCfg.ForceVerify {
		var cnt int64
		s.DB.Model(&models.User{}).Where("email = ?", req.Email).Count(&cnt)
		if cnt > 0 {
			return fmt.Errorf("该邮箱已被注册")
		}
	}
	if req.Purpose == "forgot-password" || req.Purpose == "login" {
		var cnt int64
		s.DB.Model(&models.User{}).Where("email = ?", req.Email).Count(&cnt)
		if cnt == 0 {
			if req.Purpose == "forgot-password" {
				return nil // 安全考虑：不暴露邮箱是否注册，静默返回
			}
			return fmt.Errorf("该邮箱未注册")
		}
	}

	// 频率限制：60s
	var rateCnt int64
	s.DB.Model(&models.EmailVerification{}).
		Where("email = ? AND purpose = ? AND created_at > ?", req.Email, req.Purpose, time.Now().Add(-60*time.Second)).
		Count(&rateCnt)
	if rateCnt > 0 {
		return fmt.Errorf("发送过于频繁，请60秒后再试")
	}

	code := GenerateCode()
	expires := time.Now().Add(10 * time.Minute)
	ver := models.EmailVerification{
		Email:     req.Email,
		Code:      code,
		Purpose:   req.Purpose,
		ExpiresAt: expires,
	}
	if err := s.DB.Create(&ver).Error; err != nil {
		return fmt.Errorf("存储验证码失败")
	}

	// 异步发送邮件，避免 SMTP 延迟阻塞接口响应
	go func(email, code, purpose string, cfg EmailConfig) {
		if err := SendVerificationCode(cfg, email, code, purpose); err != nil {
			slog.Error("异步发送验证码邮件失败", "email", email, "err", err)
		}
	}(req.Email, code, req.Purpose, emailCfg)
	return nil
}

// VerifyCodeRequest 验证验证码
type VerifyCodeRequest struct {
	Email   string `json:"email"`
	Code    string `json:"code"`
	Purpose string `json:"purpose"`
}

// VerifyCode 仅验证有效性，不标记已用（对齐原 /verify-code 行为）
func (s *AuthService) VerifyCode(req VerifyCodeRequest) error {
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	var v models.EmailVerification
	err := s.DB.Where("email = ? AND code = ? AND purpose = ? AND used = false AND expires_at > ?",
		req.Email, req.Code, req.Purpose, time.Now()).
		Order("id DESC").First(&v).Error
	if err != nil {
		return fmt.Errorf("验证码错误或已过期")
	}
	return nil
}

// consumeCode 验证并消费验证码（标记已用）。
// 事务内行锁 + used=false 条件更新，保证并发下同一验证码只能成功一次。
func consumeCode(db *gorm.DB, email, code, purpose string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	return db.Transaction(func(tx *gorm.DB) error {
		var v models.EmailVerification
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("email = ? AND code = ? AND purpose = ? AND used = false AND expires_at > ?",
				email, code, purpose, time.Now()).
			Order("id DESC").First(&v).Error
		if err != nil {
			return fmt.Errorf("验证码错误或已过期")
		}
		res := tx.Model(&models.EmailVerification{}).
			Where("id = ? AND used = ?", v.ID, false).
			Update("used", true)
		if res.Error != nil {
			return fmt.Errorf("验证码校验失败")
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("验证码错误或已过期")
		}
		return nil
	})
}

// ====== 免密登录 ======

// PasswordlessLoginRequest 验证码免密登录
type PasswordlessLoginRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// PasswordlessLogin 邮箱验证码免密登录
func (s *AuthService) PasswordlessLogin(req PasswordlessLoginRequest) (*LoginResult, error) {
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	emailCfg, _ := GetEmailConfig(s.DB)
	if !emailCfg.EnablePasswordless {
		return nil, fmt.Errorf("邮箱验证码登录功能未开启")
	}
	if err := consumeCode(s.DB, req.Email, req.Code, "login"); err != nil {
		return nil, err
	}
	var user models.User
	if err := s.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return nil, fmt.Errorf("该邮箱未注册")
	}
	if !user.EmailVerified {
		s.DB.Model(&user).Update("email_verified", true)
	}
	token, err := s.generateToken(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, fmt.Errorf("令牌生成失败")
	}
	return &LoginResult{
		Message: "登录成功",
		Token:   token,
		User:    toUserBrief(s.DB, &user),
	}, nil
}

// ====== 重置密码 ======

// ResetPasswordRequest 重置密码
type ResetPasswordRequest struct {
	Email    string `json:"email"`
	Code     string `json:"code"`
	Password string `json:"password"`
}

// ResetPassword 通过验证码重置密码
func (s *AuthService) ResetPassword(req ResetPasswordRequest) error {
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if len(req.Password) < 6 {
		return fmt.Errorf("密码至少需要6个字符")
	}
	if req.Code == "" {
		return fmt.Errorf("请提供验证码")
	}
	if err := consumeCode(s.DB, req.Email, req.Code, "forgot-password"); err != nil {
		return err
	}
	var user models.User
	if err := s.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return fmt.Errorf("该邮箱未注册")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		return fmt.Errorf("密码加密失败")
	}
	return s.DB.Model(&user).Update("password", string(hashed)).Error
}

// ====== 内部工具 ======

func (s *AuthService) generateToken(userID uint, username, role string) (string, error) {
	if s.JWTSecret == "" {
		return "", fmt.Errorf("JWT secret 未配置")
	}
	return util.GenerateToken(s.JWTSecret, userID, username, role)
}
