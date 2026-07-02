package router

import (
	"net/http"
	"os"
	"path/filepath"

	"xzji/backend/internal/app"
	"xzji/backend/internal/handler"
	"xzji/backend/internal/middleware"
	"xzji/backend/internal/util"

	"github.com/gin-gonic/gin"
)

func SetupInstallOnly() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), middleware.CORS())
	r.Use(gin.Logger())

	api := r.Group("/api")
	{
		api.GET("/install/status", handler.InstallStatus)
		api.POST("/install", handler.Install)
		api.GET("/health", handler.Health)
	}
	registerSPAFallback(r)
	return r
}

func SetupFull(application *app.App) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.RedirectTrailingSlash = true
	r.RedirectFixedPath = true
	r.Use(gin.Recovery(), middleware.CORS())
	r.Use(gin.Logger())

	// 将 App 实例存储到 gin.Context，供 handler 使用
	r.Use(func(c *gin.Context) {
		c.Set("app", application)
		c.Next()
	})

	api := r.Group("/api")
	{
		api.GET("/health", handler.Health)
		api.GET("/install/status", handler.InstallStatus)

		// 公开内容
		api.POST("/search", handler.PublicSearch)
		api.GET("/page/:slug", handler.PublicPage)
		api.GET("/pages", handler.PublicPages)
		api.GET("/keywords", handler.PublicKeywords)
		api.GET("/announcements/active", handler.PublicAnnouncements)
		api.GET("/categories", handler.PublicCategories)
		api.GET("/settings", handler.PublicSettings)
		api.GET("/site/config", handler.PublicSiteConfig)

		// 认证（公开 + 限流）
		auth := api.Group("/auth")
		{
			auth.POST("/login", middleware.RateLimitLogin(), handler.Login)
			auth.POST("/register", middleware.RateLimitRegister(), handler.Register)
			auth.POST("/send-code", middleware.RateLimitSendCode(), handler.SendCode)
			auth.POST("/verify-code", handler.VerifyCode)
			auth.POST("/passwordless-login", middleware.RateLimitPassword(), handler.PasswordlessLogin)
			auth.POST("/forgot-password", handler.ForgotPassword) // 对齐 Node：返回引导 JSON
			auth.POST("/reset-password", middleware.RateLimitPassword(), handler.ResetPassword)
			auth.POST("/logout", handler.Logout)
		}

		// 用户（需登录 + CSRF）
		user := api.Group("/user")
		user.Use(middleware.AuthRequired(application.JWTSecret))
		user.Use(middleware.CSRF())
		{
			user.GET("/profile", handler.GetProfile)
			user.PUT("/profile", handler.UpdateProfile)
			user.PUT("/password", handler.ChangePassword)
			user.POST("/avatar", handler.UploadAvatar)
			// 经济
			user.GET("/payment-methods", handler.PaymentMethods)
			user.POST("/recharge", handler.Recharge)
			user.GET("/recharge-status/:orderNo", handler.RechargeStatus)
			user.POST("/purchase-link", handler.PurchaseLink)
			user.GET("/purchase-status", handler.PurchaseStatus)
			user.GET("/membership", handler.GetMembership)
			user.POST("/buy-membership", handler.BuyMembership)
			user.GET("/transactions", handler.GetTransactions)
			user.GET("/downloads-today", handler.DownloadsToday)
			user.GET("/checkin-status", handler.CheckInStatus)
			user.POST("/checkin", handler.CheckIn)
			user.GET("/level-info", handler.LevelInfo)
		}

		// 支付回调（第三方支付异步/同步回调，公开）
		api.POST("/payment/callback/wechat", handler.PaymentCallbackWechat)
		api.POST("/payment/callback/alipay", handler.PaymentCallbackAlipay)
		api.GET("/payment/callback/alipay", func(c *gin.Context) { c.Redirect(http.StatusFound, "/?from=alipay") })
		api.Any("/payment/callback/epay", handler.PaymentCallbackEpay)

		// 管理（需 admin + CSRF）
		admin := api.Group("/admin")
		admin.Use(middleware.AuthRequired(application.JWTSecret), middleware.RequireAdmin())
		admin.Use(middleware.CSRF())
		{
			// 关键词
			admin.GET("/keywords", handler.AdminKeywordsList)
			admin.GET("/keywords/:id", handler.AdminKeywordGet)
			admin.POST("/keywords", handler.AdminKeywordCreate)
			admin.PUT("/keywords/:id", handler.AdminKeywordUpdate)
			admin.DELETE("/keywords/:id", handler.AdminKeywordDelete)
			admin.GET("/keywords/:id/categories", handler.AdminKeywordCategoriesGet)
			admin.PUT("/keywords/:id/categories", handler.AdminKeywordCategoriesSet)
			// 分类
			admin.GET("/categories", handler.AdminCategoriesList)
			admin.POST("/categories", handler.AdminCategoryCreate)
			admin.PUT("/categories/:id", handler.AdminCategoryUpdate)
			admin.DELETE("/categories/:id", handler.AdminCategoryDelete)
			// 页面
			admin.GET("/pages", handler.AdminPagesList)
			admin.POST("/pages", handler.AdminPageCreate)
			admin.PUT("/pages/:id", handler.AdminPageUpdate)
			admin.DELETE("/pages/:id", handler.AdminPageDelete)
			// 公告
			admin.GET("/announcements", handler.AdminAnnouncementsList)
			admin.POST("/announcements", handler.AdminAnnouncementCreate)
			admin.PUT("/announcements/:id", handler.AdminAnnouncementUpdate)
			admin.DELETE("/announcements/:id", handler.AdminAnnouncementDelete)
			// 支付订单
			admin.GET("/payment-orders", handler.AdminPaymentOrdersList)
			admin.GET("/payment-orders/stats", handler.AdminPaymentOrdersStats)
			admin.POST("/payment-orders/batch-delete", handler.AdminPaymentOrderBatchDelete)
			admin.DELETE("/payment-orders/:id", handler.AdminPaymentOrderDelete)
			// 仪表盘 + 配置
			admin.GET("/stats", handler.AdminStats)
			admin.GET("/settings", handler.AdminSettingsGet)
			admin.PUT("/settings", handler.AdminSettingsPut)
			admin.POST("/upload-logo", handler.AdminUploadLogo)
			admin.GET("/email-config", handler.AdminEmailConfigGet)
			admin.PUT("/email-config", handler.AdminEmailConfigPut)
			admin.POST("/email-test", handler.AdminEmailTest)
			admin.GET("/checkin-config", handler.AdminCheckinConfigGet)
			admin.PUT("/checkin-config", handler.AdminCheckinConfigPut)
			admin.GET("/level-config", handler.AdminLevelConfigGet)
			admin.PUT("/level-config", handler.AdminLevelConfigPut)
			admin.GET("/membership-config", handler.AdminMembershipConfigGet)
			admin.PUT("/membership-config", handler.AdminMembershipConfigPut)
			admin.GET("/payment-config", handler.AdminPaymentConfigGet)
			admin.PUT("/payment-config", handler.AdminPaymentConfigPut)
			// 会员到期提醒（手动触发）
			admin.POST("/trigger-membership-reminders", handler.TriggerMembershipReminders)
			// 用户管理
			admin.GET("/users", handler.AdminUsersList)
			admin.GET("/users/:id", handler.AdminUserGet)
			admin.PUT("/users/:id", handler.AdminUserUpdate)
			admin.DELETE("/users/:id", handler.AdminUserDelete)
		}
	}

	// 上传目录静态服务
	uploadsDir := filepath.Join(backendRoot(), "uploads")
	_ = os.MkdirAll(uploadsDir, 0755)
	r.Static("/uploads", uploadsDir)

	registerSPAFallback(r)
	return r
}

func registerSPAFallback(r *gin.Engine) {
	candidates := []string{
		filepath.Join(backendRoot(), "public"),
		filepath.Join(backendRoot(), "..", "frontend", "dist"),
	}
	var distDir string
	for _, p := range candidates {
		if fi, err := os.Stat(filepath.Join(p, "index.html")); err == nil && !fi.IsDir() {
			distDir = p
			break
		}
	}
	if distDir == "" {
		return
	}
	r.Static("/assets", filepath.Join(distDir, "assets"))
	r.StaticFile("/favicon.ico", filepath.Join(distDir, "favicon.ico"))
	r.Use(func(c *gin.Context) {
		p := c.Request.URL.Path
		if len(p) >= 4 && p[:4] == "/api" {
			c.Next()
			return
		}
		if c.Request.Method == http.MethodGet {
			fp := filepath.Join(distDir, p)
			if fi, err := os.Stat(fp); err == nil && !fi.IsDir() {
				c.File(fp)
				c.Abort()
				return
			}
			c.File(filepath.Join(distDir, "index.html"))
			c.Abort()
			return
		}
		c.Next()
	})
}

func backendRoot() string { return util.BackendRoot() }
