package database

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"xzji/backend/internal/config"
	"xzji/backend/internal/models"
	"xzji/backend/internal/util"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

// DB 返回全局 GORM 实例（已安装后调用）
func DB() *gorm.DB {
	return db
}

// Init 初始化连接池并执行迁移（已安装阶段调用）
func Init(cfg *config.Config) error {
	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	}
	gdb, err := gorm.Open(mysql.Open(cfg.Database.DSN()), gormCfg)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return fmt.Errorf("获取底层 sql.DB 失败: %w", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := models.Migrate(gdb); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}
	if err := Seed(gdb); err != nil {
		return fmt.Errorf("种子数据写入失败: %w", err)
	}
	// 运行 SQL 文件增量迁移（schema_migrations 版本管理）
	migrationsDir := filepath.Join(util.BackendRoot(), "migrations")
	if err := RunMigrations(gdb, migrationsDir); err != nil {
		return fmt.Errorf("增量迁移执行失败: %w", err)
	}
	db = gdb
	slog.Info("数据库连接成功", "host", cfg.Database.Host, "db", cfg.Database.Name)
	return nil
}

// CreateDatabase 连接到 MySQL（不指定库）并创建目标库。安装向导阶段调用。
func CreateDatabase(host string, port int, user, password, dbName string) error {
	if !isSafeDatabaseName(dbName) {
		return fmt.Errorf("数据库名称仅允许英文字母、数字和下划线")
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local",
		user, password, host, port)
	gdb, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		return fmt.Errorf("连接 MySQL 失败: %w", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	// 使用 utf8mb4 字符集建库
	sql := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName)
	if err := gdb.Exec(sql).Error; err != nil {
		return fmt.Errorf("创建数据库失败: %w", err)
	}
	slog.Info("数据库已创建（或已存在）", "db", dbName)
	return nil
}

func isSafeDatabaseName(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return false
	}
	return true
}

// SetupFromInstall 安装向导专用：建库 → 连库 → 建表 → 种子
func SetupFromInstall(host string, port int, user, password, dbName string) error {
	if err := CreateDatabase(host, port, user, password, dbName); err != nil {
		return err
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, password, host, port, dbName)
	gdb, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		return fmt.Errorf("连接新库失败: %w", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	if err := models.Migrate(gdb); err != nil {
		return fmt.Errorf("建表失败: %w", err)
	}
	if err := Seed(gdb); err != nil {
		return fmt.Errorf("种子数据失败: %w", err)
	}
	slog.Info("安装：建表与种子数据完成", "db", dbName)
	return nil
}
