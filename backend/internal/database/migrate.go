package database

import (
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

// RunMigrations 读取 migrations/ 目录下的 *.up.sql 文件，按文件名排序，
// 将未执行过的迁移依次应用到数据库，每执行一条就在 schema_migrations 表中记录版本。
//
// 首次运行时（schema_migrations 为空），将当前 AutoMigrate 建立的表结构标记为 baseline，
// 后续新增的 SQL 迁移文件从下一个版本号开始递增。
func RunMigrations(db *gorm.DB, migrationsDir string) error {
	// 确保迁移版本表存在
	if err := db.AutoMigrate(&models.SchemaMigration{}); err != nil {
		return fmt.Errorf("创建迁移版本表失败: %w", err)
	}

	// 首次运行：标记 baseline（当前 AutoMigrate 建立的表结构即为基线）
	var count int64
	db.Model(&models.SchemaMigration{}).Count(&count)
	if count == 0 {
		db.Create(&models.SchemaMigration{Version: "000001_baseline", AppliedAt: time.Now()})
		slog.Info("数据库迁移基线已标记", "version", "000001_baseline")
	}

	// 扫描迁移文件
	pattern := filepath.Join(migrationsDir, "*.up.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("扫描迁移目录失败: %w", err)
	}
	sort.Strings(files)

	for _, f := range files {
		version := strings.TrimSuffix(filepath.Base(f), ".up.sql")

		// 跳过已应用的版本
		var existing models.SchemaMigration
		if err := db.Where("version = ?", version).First(&existing).Error; err == nil {
			continue
		}

		sql, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("读取迁移文件 %s 失败: %w", f, err)
		}
		sqlStr := strings.TrimSpace(string(sql))
		if sqlStr == "" {
			slog.Warn("迁移文件为空，跳过", "file", f)
			continue
		}

		slog.Info("正在应用数据库迁移", "version", version)
		if err := db.Exec(sqlStr).Error; err != nil {
			return fmt.Errorf("迁移 %s 执行失败: %w", version, err)
		}

		db.Create(&models.SchemaMigration{Version: version, AppliedAt: time.Now()})
		slog.Info("数据库迁移已应用", "version", version)
	}

	return nil
}
