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

		// 授权停服守卫：授权无效时拦截前台业务接口（接入文档「停止对外服务」约定：
		// 前台停服，管理后台/登录/站点设置/支付回调保持可用）
		guard := middleware.LicenseGuard(application.License)

		// 公开内容（受授权守卫）
		pub := api.Group("", guard)
		{
			pub.POST("/search", handler.PublicSearch)
			pub.GET("/page/:slug", handler.PublicPage)
			pub.GET("/pages", handler.PublicPages)
			pub.GET("/keywords", handler.PublicKeywords)
			pub.GET("/keywords/featured", handler.PublicFeaturedKeywords)
			pub.GET("/announcements/active", handler.PublicAnnouncements)
			pub.GET("/categories", handler.PublicCategories)
		}
		// 站点设置/授权状态：放行，前端渲染框架与停服遮罩依赖它们
		api.GET("/settings", handler.PublicSettings)
		api.GET("/site/config", handler.PublicSiteConfig)
		api.GET("/license/status", handler.PublicLicenseStatus)

		// 认证（公开 + 限流）。登录/找回密码放行（管理员需要登录后台处理授权），
		// 注册属于业务功能，停服期间拦截。
		auth := api.Group("/auth")
		{
			auth.POST("/login", middleware.RateLimitLogin(), handler.Login)
			auth.POST("/register", guard, middleware.RateLimitRegister(), handler.Register)
			auth.POST("/send-code", middleware.RateLimitSendCode(), handler.SendCode)
			auth.POST("/verify-code", handler.VerifyCode)
			auth.POST("/passwordless-login", middleware.RateLimitPassword(), handler.PasswordlessLogin)
			auth.POST("/forgot-password", handler.ForgotPassword) // 对齐 Node：返回引导 JSON
			auth.POST("/reset-password", middleware.RateLimitPassword(), handler.ResetPassword)
			auth.POST("/logout", handler.Logout)
		}

		// 用户（需登录 + CSRF）。GET /profile 放行：管理员会话恢复（前端路由守卫）依赖；
		// 其余用户业务接口停服期间拦截。
		user := api.Group("/user")
		user.Use(middleware.AuthRequired(application.JWTSecret))
		user.Use(middleware.CSRF())
		user.GET("/profile", handler.GetProfile)
		userBiz := user.Group("", guard)
		{
			userBiz.PUT("/profile", handler.UpdateProfile)
			userBiz.PUT("/password", handler.ChangePassword)
			userBiz.POST("/avatar", handler.UploadAvatar)
			// 经济
			userBiz.GET("/payment-methods", handler.PaymentMethods)
			userBiz.POST("/recharge", handler.Recharge)
			userBiz.GET("/recharge-status/:orderNo", handler.RechargeStatus)
			userBiz.POST("/purchase-link", handler.PurchaseLink)
			userBiz.GET("/purchase-status", handler.PurchaseStatus)
			userBiz.GET("/membership", handler.GetMembership)
			userBiz.POST("/buy-membership", handler.BuyMembership)
			userBiz.GET("/transactions", handler.GetTransactions)
			userBiz.GET("/downloads-today", handler.DownloadsToday)
			// 收藏
			userBiz.POST("/favorites/:keywordId", handler.ToggleFavorite)
			userBiz.GET("/favorites", handler.ListFavorites)
			userBiz.GET("/favorites/ids", handler.ListFavoriteIDs)
			userBiz.GET("/checkin-status", handler.CheckInStatus)
			userBiz.POST("/checkin", handler.CheckIn)
			userBiz.GET("/level-info", handler.LevelInfo)
		}

		// 支付回调（第三方支付异步/同步回调，公开）。
		// 停服期间仍放行：用户已付款的订单必须正常入账，避免资金纠纷。
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
			// 授权管理（授权升级中心接入：状态/激活/心跳/升级）
			admin.GET("/license", handler.AdminLicenseGet)
			admin.POST("/license/activate", handler.AdminLicenseActivate)
			admin.POST("/license/heartbeat", handler.AdminLicenseHeartbeat)
			admin.POST("/license/upgrade", handler.AdminLicenseUpgrade)
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
