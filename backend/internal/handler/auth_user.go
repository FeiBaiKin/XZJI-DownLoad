package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"xzji/backend/internal/middleware"
	"xzji/backend/internal/models"
	"xzji/backend/internal/service"
	"xzji/backend/internal/util"

	"github.com/gin-gonic/gin"
)

// authSvc 从 context 获取 AuthService（依赖注入）
func authSvc(c *gin.Context) *service.AuthService {
	return getApp(c).AuthService
}

// userSvc 从 context 获取 UserService（依赖注入）
func userSvc(c *gin.Context) *service.UserService {
	return getApp(c).UserService
}

// uploadsDir 解析 backend/uploads，与 router 的 /uploads 静态服务一致：
// 优先工作目录(含 go.mod，兼容 go run)，其次可执行文件目录(生产环境)。
// 避免 handler 用裸 os.Getwd() 与 router 的 backendRoot() 解析不一致导致文件存取目录错位。
func uploadsDir() string {
	dir := backendDir()
	uploads := filepath.Join(dir, "uploads")
	os.MkdirAll(uploads, 0755)
	return uploads
}

// backendDir 返回 backend 根目录，统一使用 util.BackendRoot。
func backendDir() string { return util.BackendRoot() }

func setAuthCookie(c *gin.Context, token string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		Secure:   middleware.SecureCookie(c),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// ---- 认证 ----

func Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供用户名和密码"})
		return
	}
	res, err := authSvc(c).Login(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	setAuthCookie(c, res.Token, 86400)
	c.JSON(http.StatusOK, res)
}

func Register(c *gin.Context) {
	var req service.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}
	res, err := authSvc(c).Register(req)
	if err != nil {
		code := http.StatusBadRequest
		msg := err.Error()
		if msg == "用户名已存在" || msg == "邮箱已被注册" {
			code = http.StatusConflict
		}
		c.JSON(code, gin.H{"error": msg})
		return
	}
	setAuthCookie(c, res.Token, 86400)
	c.JSON(http.StatusCreated, res)
}

func SendCode(c *gin.Context) {
	var req service.SendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}
	err := authSvc(c).SendCode(req)
	if err != nil {
		msg := err.Error()
		// "forgot-password 且邮箱不存在" 时 service 返回 nil（安全考虑不暴露）
		// 但 SMTP 未配置等真实错误需要返回
		if msg == "" || msg == "发送成功" {
			c.JSON(http.StatusOK, gin.H{"message": "如果该邮箱已注册，验证码已发送", "expiresIn": 600})
			return
		}
		status := http.StatusBadRequest
		if msg == "发送过于频繁，请60秒后再试" {
			status = http.StatusTooManyRequests
		}
		// SMTP 未配置等
		c.JSON(status, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "验证码已发送", "expiresIn": 600})
}

func VerifyCode(c *gin.Context) {
	var req service.VerifyCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}
	if err := authSvc(c).VerifyCode(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"verified": true})
}

func PasswordlessLogin(c *gin.Context) {
	var req service.PasswordlessLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}
	res, err := authSvc(c).PasswordlessLogin(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	setAuthCookie(c, res.Token, 86400)
	c.JSON(http.StatusOK, res)
}

func ResetPassword(c *gin.Context) {
	var req service.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}
	if err := authSvc(c).ResetPassword(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "密码重置成功，请使用新密码登录"})
}

// Logout POST /api/auth/logout
// 清除 HttpOnly JWT cookie 和 CSRF cookie
func Logout(c *gin.Context) {
	setAuthCookie(c, "", -1)
	middleware.SetCSRFCookie(c, "", -1)
	c.JSON(http.StatusOK, gin.H{"message": "已退出登录"})
}

// ForgotPassword POST /api/auth/forgot-password
// 对齐 Node 原版语义：返回引导 JSON，提示用户使用 send-code + reset-password 流程。
// 不实际发送验证码（由前端引导用户调用 /send-code）。
func ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供注册邮箱"})
		return
	}
	// 不暴露邮箱是否已注册（安全考虑），统一返回引导信息
	c.JSON(http.StatusOK, gin.H{
		"message":     "请点击发送验证码获取验证码，然后输入验证码和新密码完成重置",
		"useSendCode": true,
	})
}

// ---- 用户 ----

func GetProfile(c *gin.Context) {
	brief, err := userSvc(c).GetProfile(middleware.GetUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, brief)
}

func UpdateProfile(c *gin.Context) {
	var req service.ProfileUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}
	brief, err := userSvc(c).UpdateProfile(middleware.GetUserID(c), req)
	if err != nil {
		code := http.StatusBadRequest
		if err.Error() == "邮箱已被其他用户使用" {
			code = http.StatusConflict
		}
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, brief)
}

func ChangePassword(c *gin.Context) {
	var req service.PasswordChange
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}
	if err := userSvc(c).ChangePassword(middleware.GetUserID(c), req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "密码修改成功"})
}

// DownloadsToday GET /api/user/downloads-today
// 今日已下载次数（对齐原项目 user.js:472-484）。
func DownloadsToday(c *gin.Context) {
	uid := middleware.GetUserID(c)
	var user models.User
	if err := getApp(c).DB.Select("daily_downloads, download_date").Where("id = ?", uid).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取下载次数失败"})
		return
	}
	// 日界线用本地时区，与 incrementDailyDownloads 写入口径一致；DATE 列直接 Format（禁用 .UTC()）
	today := time.Now().Format("2006-01-02")
	downloads := 0
	if user.DownloadDate != nil && user.DownloadDate.Format("2006-01-02") == today {
		downloads = user.DailyDownloads
	}
	c.JSON(http.StatusOK, gin.H{"downloads": downloads, "date": today})
}

func UploadAvatar(c *gin.Context) {
	// 限制上传大小 5MB
	const maxUploadSize = 5 << 20
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

	file, err := c.FormFile("avatar")
	if err != nil {
		if err.Error() == "http: request body too large" {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "文件大小不能超过 5MB"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要上传的头像文件"})
		return
	}

	// 文件大小二次校验
	if file.Size > maxUploadSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "文件大小不能超过 5MB"})
		return
	}

	// 扩展名校验（大小写不敏感）
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
	if !allowedExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 JPG、PNG、GIF、WebP 格式的图片"})
		return
	}

	uid := middleware.GetUserID(c)
	filename := fmt.Sprintf("avatar_%d_%d%s", uid, time.Now().Unix(), ext)
	savePath := filepath.Join(uploadsDir(), filename)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "头像上传失败"})
		return
	}
	avatarURL := "/uploads/" + filename
	if err := userSvc(c).UpdateAvatar(uid, avatarURL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "头像上传失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "头像上传成功", "avatar": avatarURL})
}

// AdminUploadLogo POST /api/admin/upload-logo
// 站点 Logo 上传，保存到 uploads 并写入 site_settings.siteLogoUrl
func AdminUploadLogo(c *gin.Context) {
	// 限制上传大小 5MB
	const maxUploadSize = 5 << 20
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

	file, err := c.FormFile("logo")
	if err != nil {
		if err.Error() == "http: request body too large" {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "文件大小不能超过 5MB"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要上传的 Logo 文件"})
		return
	}

	// 文件大小二次校验
	if file.Size > maxUploadSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "文件大小不能超过 5MB"})
		return
	}

	// 扩展名校验（大小写不敏感）
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".svg": true}
	if !allowedExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 JPG、PNG、GIF、WebP、SVG 格式的图片"})
		return
	}

	filename := fmt.Sprintf("logo_%d%s", time.Now().Unix(), ext)
	savePath := filepath.Join(uploadsDir(), filename)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Logo 上传失败"})
		return
	}
	url := "/uploads/" + filename
	if err := service.UpdateSiteSettings(getApp(c).DB, map[string]string{"siteLogoUrl": url}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Logo 保存失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url, "message": "Logo 上传成功"})
}
