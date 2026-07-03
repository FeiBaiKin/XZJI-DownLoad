package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"xzji/backend/internal/app"
	"xzji/backend/internal/config"
	"xzji/backend/internal/database"
	"xzji/backend/internal/license"
	"xzji/backend/internal/middleware"
	"xzji/backend/internal/router"
	"xzji/backend/internal/scheduler"
	"xzji/backend/internal/service"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	installed := service.IsInstalled()

	if !installed {
		fmt.Println()
		fmt.Println("  ╔══════════════════════════════════════╗")
		fmt.Println("  ║       系统尚未安装，请先完成配置      ║")
		fmt.Println("  ║                                      ║")
		fmt.Println("  ║  访问 http://localhost:4000/install  ║")
		fmt.Println("  ║  完成安装后重启服务即可使用           ║")
		fmt.Println("  ╚══════════════════════════════════════╝")
		fmt.Println()
		r := router.SetupInstallOnly()
		srv := &http.Server{Addr: ":4000", Handler: r}
		go gracefulShutdown(srv, "安装服务", nil)
		slog.Info("安装服务已启动", "port", 4000)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("安装服务启动失败", "err", err)
			os.Exit(1)
		}
		return
	}

	// 已安装 → 加载配置 → 初始化数据库 → 注入 JWT
	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "err", err)
		os.Exit(1)
	}

	// 注入 JWT Secret 到全局（middleware + service 共享）
	jwtSecret := cfg.JWT.Secret
	if jwtSecret == "" {
		slog.Warn("JWT secret 为空，使用随机生成（仅开发安全）")
		jwtSecret = config.GenerateJWTSecret()
	}

	// 注入 CORS 允许来源
	if len(cfg.Server.AllowedOrigins) > 0 {
		middleware.AllowedOrigins = cfg.Server.AllowedOrigins
		slog.Info("CORS 已配置", "allowedOrigins", cfg.Server.AllowedOrigins)
	}

	if err := database.Init(cfg); err != nil {
		slog.Error("数据库初始化失败", "err", err)
		os.Exit(1)
	}

	// 启动会员到期提醒调度器（5s 后首次检查，之后每 24h）
	scheduler.Start(database.DB())

	// 授权升级中心客户端：加载本地缓存凭证并启动心跳循环
	// （域名取安装向导写入 allowed_origins 的站点地址）
	siteURL := ""
	if len(cfg.Server.AllowedOrigins) > 0 {
		siteURL = cfg.Server.AllowedOrigins[0]
	}
	licenseMgr := license.NewManager(siteURL)
	licenseMgr.Load()
	licenseMgr.Start()

	// 创建应用实例（依赖注入）
	application := app.NewApp(database.DB(), jwtSecret, licenseMgr)

	port := cfg.Server.Port
	if port == 0 {
		port = 4000
	}
	r := router.SetupFull(application)
	srv := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: r}
	go gracefulShutdown(srv, "后端服务", licenseMgr)
	slog.Info("后端服务已启动", "port", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("服务启动失败", "err", err)
		os.Exit(1)
	}
}

// gracefulShutdown 监听系统信号，优雅关闭 HTTP 服务、调度器、授权心跳和数据库连接池。
func gracefulShutdown(srv *http.Server, name string, lic *license.Manager) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("收到关闭信号，正在优雅退出...", "signal", sig.String(), "service", name)

	// 15 秒超时等待所有活跃请求完成
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 停止调度器与授权心跳循环
	scheduler.Stop()
	if lic != nil {
		lic.Stop()
	}

	// 关闭 HTTP 服务（不再接受新连接，等待现有请求完成）
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("HTTP 服务关闭异常", "err", err)
	}

	// 关闭数据库连接池
	if db := database.DB(); db != nil {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
			slog.Info("数据库连接池已关闭")
		}
	}

	slog.Info("服务已安全退出", "service", name)
}
