package service

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"xzji/backend/internal/models"
)

// UserService 用户功能业务逻辑
type UserService struct {
	DB *gorm.DB
}

// ====== 资料 ======

// ProfileUpdate 更新资料请求
type ProfileUpdate struct {
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
}

// GetProfile 获取当前用户资料
func (s *UserService) GetProfile(userID uint) (*UserBrief, error) {
	var user models.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("用户不存在")
	}
	b := toUserBrief(s.DB, &user)
	return &b, nil
}

// UpdateProfile 更新资料
func (s *UserService) UpdateProfile(userID uint, req ProfileUpdate) (*UserBrief, error) {
	nickname := strings.TrimSpace(req.Nickname)
	email := strings.TrimSpace(strings.ToLower(req.Email))
	updates := map[string]any{}
	if nickname != "" {
		updates["nickname"] = nickname
	}
	if email != "" {
		var cnt int64
		s.DB.Model(&models.User{}).Where("email = ? AND id != ?", email, userID).Count(&cnt)
		if cnt > 0 {
			return nil, fmt.Errorf("邮箱已被其他用户使用")
		}
		updates["email"] = email
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("没有提供需要更新的字段")
	}
	if err := s.DB.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新失败")
	}
	return s.GetProfile(userID)
}

// ====== 密码 ======

// PasswordChange 修改密码
type PasswordChange struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// ChangePassword 修改密码
func (s *UserService) ChangePassword(userID uint, req PasswordChange) error {
	if len(req.NewPassword) < 6 {
		return fmt.Errorf("新密码至少需要6个字符")
	}
	if req.CurrentPassword == req.NewPassword {
		return fmt.Errorf("新密码不能与当前密码相同")
	}
	var user models.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		return fmt.Errorf("用户不存在")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		return fmt.Errorf("当前密码不正确")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 10)
	if err != nil {
		return fmt.Errorf("密码加密失败")
	}
	return s.DB.Model(&user).Update("password", string(hashed)).Error
}

// ====== 头像 ======

// UpdateAvatar 更新头像路径
func (s *UserService) UpdateAvatar(userID uint, avatarURL string) error {
	return s.DB.Model(&models.User{}).Where("id = ?", userID).Update("avatar", avatarURL).Error
}
