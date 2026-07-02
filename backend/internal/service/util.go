package service

import (
	"os"
	"path/filepath"
	"time"

	"xzji/backend/internal/util"

	"gorm.io/driver/mysql"
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

// mysqlDialector 返回 MySQL Dialector
func mysqlDialector(dsn string) gorm.Dialector {
	return mysql.Open(dsn)
}
