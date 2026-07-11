package service

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"xzji/backend/internal/util"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// installLockPath 安装锁文件路径：backend/.installed
func installLockPath() string {
	return filepath.Join(util.BackendRoot(), ".installed")
}

// installLockExists 安装锁是否存在
func installLockExists() bool {
	_, err := os.Stat(installLockPath())
	return err == nil
}

// writeInstallLockImpl 写入安装锁文件
func writeInstallLockImpl(t time.Time) error {
	content := t.UTC().Format(time.RFC3339)
	return os.WriteFile(installLockPath(), []byte(content), 0644)
}

// postgresDialector 返回 PostgreSQL Dialector。
func postgresDialector(dsn string) gorm.Dialector {
	return postgres.Open(dsn)
}

// isUniqueViolation 识别 PostgreSQL unique_violation（SQLSTATE 23505）。
func isUniqueViolation(err error) bool {
	type sqlStateError interface{ SQLState() string }
	var stateErr sqlStateError
	return errors.As(err, &stateErr) && stateErr.SQLState() == "23505"
}
