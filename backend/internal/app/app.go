package app

import (
	"gorm.io/gorm"
	"xzji/backend/internal/cache"
	"xzji/backend/internal/license"
	"xzji/backend/internal/service"
)

// App 持有应用所有依赖，通过依赖注入传递给 router/handler/middleware。
// 消除全局变量（middleware.JWTSecret / service.JWTSecret / database.DB()），
// 使代码可测试、可维护。
type App struct {
	DB        *gorm.DB
	JWTSecret string

	// Services（按需创建，持有 DB 引用）
	AuthService *service.AuthService
	UserService *service.UserService
	ContentSvc  *service.ContentService
	EconomySvc  *service.EconomyService
	FavoriteSvc *service.FavoriteService

	// Cache 配置缓存
	Cache *cache.ConfigCache

	// License 授权升级中心客户端（验签状态机 + 心跳循环）
	License *license.Manager
}

// NewApp 创建 App 实例，初始化所有 services。
func NewApp(db *gorm.DB, jwtSecret string, lic *license.Manager) *App {
	return &App{
		DB:          db,
		JWTSecret:   jwtSecret,
		AuthService: &service.AuthService{DB: db, JWTSecret: jwtSecret},
		UserService: &service.UserService{DB: db},
		ContentSvc:  &service.ContentService{DB: db},
			EconomySvc:  &service.EconomyService{DB: db},
			FavoriteSvc: &service.FavoriteService{DB: db},
			Cache:       cache.NewConfigCache(),
		License:     lic,
	}
}
