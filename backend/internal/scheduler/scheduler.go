package scheduler

import (
	"log/slog"
	"sync"
	"time"

	"gorm.io/gorm"
	"xzji/backend/internal/service"
)

var (
	stopCh chan struct{}
	once   sync.Once
)

// Start 启动会员到期提醒调度器：首次延迟 5 秒，之后每 24 小时检查一次。
// 对齐原项目 index.js：setTimeout(5s) → setInterval(24h) 调 checkMembershipExpiryReminders。
// 阻塞在独立 goroutine 中运行，不影响主服务。
func Start(db *gorm.DB) {
	stopCh = make(chan struct{})
	go func() {
		// 首次执行延迟 5s，或收到停止信号时立即退出
		select {
		case <-time.After(5 * time.Second):
		case <-stopCh:
			return
		}
		run(db)
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				run(db)
			case <-stopCh:
				return
			}
		}
	}()
	slog.Info("会员到期提醒调度器已启动", "interval", "24h", "initialDelay", "5s")
}

// Stop 停止调度器，优雅关闭时调用。
func Stop() {
	once.Do(func() {
		if stopCh != nil {
			close(stopCh)
			slog.Info("会员到期提醒调度器已停止")
		}
	})
}

func run(db *gorm.DB) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("会员到期提醒检查异常", "err", r)
		}
	}()
	sent, err := service.CheckMembershipExpiryReminders(db)
	if err != nil {
		slog.Error("会员到期提醒检查失败", "err", err)
		return
	}
	if sent > 0 {
		slog.Info("会员到期提醒调度完成", "sent", sent)
	}
}
