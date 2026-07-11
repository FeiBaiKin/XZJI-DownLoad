package database

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"xzji/backend/internal/config"
	"xzji/backend/internal/util"

	"gorm.io/driver/postgres"
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
	gdb, err := gorm.Open(postgres.Open(cfg.Database.DSN()), gormCfg)
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

	// 运行 SQL 文件增量迁移（schema_migrations 版本管理）
	migrationsDir := filepath.Join(util.BackendRoot(), "migrations")
	if err := RunMigrations(gdb, migrationsDir); err != nil {
		return fmt.Errorf("增量迁移执行失败: %w", err)
	}
	db = gdb
	slog.Info("数据库连接成功", "host", cfg.Database.Host, "db", cfg.Database.Name)
	return nil
}

// CreateDatabase 连接 PostgreSQL 默认维护库并创建目标库。安装向导阶段调用。
func CreateDatabase(host string, port int, user, password, dbName, sslMode, timeZone string) error {
	if !isSafeDatabaseName(dbName) {
		return fmt.Errorf("数据库名称仅允许英文字母、数字和下划线")
	}
	maintenance := config.DatabaseConfig{Host: host, Port: port, User: user, Password: password, Name: dbName, SSLMode: sslMode, TimeZone: timeZone}
	gdb, err := gorm.Open(postgres.Open(maintenance.DSNWithoutDB()), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		return fmt.Errorf("连接 PostgreSQL 失败: %w", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	var exists bool
	if err := gdb.Raw("SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = ?)", dbName).Scan(&exists).Error; err != nil {
		return fmt.Errorf("检查数据库是否存在失败: %w", err)
	}
	if !exists {
		// 数据库名已经过严格白名单校验；双引号保留用户指定的大小写。
		sql := fmt.Sprintf(`CREATE DATABASE "%s" ENCODING 'UTF8'`, dbName)
		if err := gdb.Exec(sql).Error; err != nil && !hasSQLState(err, "42P04") {
			return fmt.Errorf("创建数据库失败: %w", err)
		}
	}
	slog.Info("数据库已创建（或已存在）", "db", dbName)
	return nil
}
func hasSQLState(err error, wanted string) bool {
	type sqlStateError interface{ SQLState() string }
	var stateErr sqlStateError
	return errors.As(err, &stateErr) && stateErr.SQLState() == wanted
}

func isSafeDatabaseName(name string) bool {
	if name == "" || len(name) > 63 {
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

// SetupFromInstall 安装向导专用：建库 → 连库 → 建表 → 写入最小安装数据
func SetupFromInstall(host string, port int, user, password, dbName, sslMode, timeZone string) error {
	if err := CreateDatabase(host, port, user, password, dbName, sslMode, timeZone); err != nil {
		return err
	}
	dbCfg := config.DatabaseConfig{Host: host, Port: port, User: user, Password: password, Name: dbName, SSLMode: sslMode, TimeZone: timeZone}
	gdb, err := gorm.Open(postgres.Open(dbCfg.DSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		return fmt.Errorf("连接新库失败: %w", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	migrationsDir := filepath.Join(util.BackendRoot(), "migrations")
	if err := RunMigrations(gdb, migrationsDir); err != nil {
		return fmt.Errorf("增量迁移执行失败: %w", err)
	}
	if err := SeedInstallData(gdb); err != nil {
		return fmt.Errorf("最小安装数据写入失败: %w", err)
	}
	slog.Info("安装：建表完成", "db", dbName)
	return nil
}
