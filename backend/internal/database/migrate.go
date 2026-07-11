package database

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"xzji/backend/internal/models"

	"gorm.io/gorm"
)

// RunMigrations 按文件名顺序应用尚未执行的 *.up.sql。
// PostgreSQL advisory transaction lock 保证多实例不会并发迁移；每个迁移的 SQL
// 与版本记录位于同一事务，任一步失败都会完整回滚。
func RunMigrations(db *gorm.DB, migrationsDir string) error {
	pattern := filepath.Join(migrationsDir, "*.up.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("扫描迁移目录失败: %w", err)
	}
	sort.Strings(files)

	return db.Transaction(func(tx *gorm.DB) error {
		// 固定键只在本应用内使用。事务结束时 PostgreSQL 会自动释放锁。
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "xzji_schema_migrations").Error; err != nil {
			return fmt.Errorf("获取数据库迁移锁失败: %w", err)
		}

		if err := models.Migrate(tx); err != nil {
			return fmt.Errorf("数据库结构迁移失败: %w", err)
		}

		var count int64
		if err := tx.Model(&models.SchemaMigration{}).Count(&count).Error; err != nil {
			return fmt.Errorf("读取迁移版本失败: %w", err)
		}
		if count == 0 {
			if err := tx.Create(&models.SchemaMigration{Version: "000001_baseline", AppliedAt: time.Now()}).Error; err != nil {
				return fmt.Errorf("标记数据库迁移基线失败: %w", err)
			}
			slog.Info("数据库迁移基线已标记", "version", "000001_baseline")
		}

		for _, f := range files {
			version := strings.TrimSuffix(filepath.Base(f), ".up.sql")
			var existing models.SchemaMigration
			err := tx.Where("version = ?", version).First(&existing).Error
			if err == nil {
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("查询迁移版本 %s 失败: %w", version, err)
			}

			sqlBytes, err := os.ReadFile(f)
			if err != nil {
				return fmt.Errorf("读取迁移文件 %s 失败: %w", f, err)
			}
			sqlStr := strings.TrimSpace(string(sqlBytes))
			if sqlStr == "" {
				slog.Warn("迁移文件为空，跳过", "file", f)
				continue
			}

			slog.Info("正在应用数据库迁移", "version", version)
			if err := tx.Exec(sqlStr).Error; err != nil {
				return fmt.Errorf("迁移 %s 执行失败: %w", version, err)
			}
			if err := tx.Create(&models.SchemaMigration{Version: version, AppliedAt: time.Now()}).Error; err != nil {
				return fmt.Errorf("记录迁移版本 %s 失败: %w", version, err)
			}
			slog.Info("数据库迁移已应用", "version", version)
		}
		return nil
	})
}
