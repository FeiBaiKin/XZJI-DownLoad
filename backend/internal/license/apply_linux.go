//go:build linux

package license

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"xzji/backend/internal/util"
	"xzji/backend/internal/version"
)

// apply Linux 生产环境：解包 → 备份 → 替换二进制/public/合并 migrations。
// 重启由调用方通过 ScheduleRestart 触发（先让 HTTP 响应发出）。
func (m *Manager) apply(pkgPath, newVersion string) (*UpgradeResult, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("定位当前可执行文件失败: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
		exePath = resolved
	}
	// go run 的临时二进制无法自替换
	if strings.Contains(exePath, "go-build") {
		return &UpgradeResult{
			Version:     newVersion,
			Applied:     false,
			StagingPath: pkgPath,
			Message:     "开发模式（go run）不支持自动替换，更新包已下载到暂存目录，请手动处理",
		}, nil
	}

	root := util.BackendRoot()
	extractDir := filepath.Join(root, "update_staging", "extract-"+newVersion)
	_ = os.RemoveAll(extractDir)
	if err := extractTarGz(pkgPath, extractDir); err != nil {
		return nil, err
	}

	newBin := findNewBinary(extractDir, filepath.Base(exePath))
	if newBin == "" {
		return nil, errors.New("更新包中未找到可执行文件（约定：包根目录含 xzji-linux-amd64 或与当前可执行文件同名的文件）")
	}

	backupDir := filepath.Join(root, "backup", fmt.Sprintf("%s-v%s", time.Now().Format("20060102-150405"), version.Version))
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建备份目录失败: %w", err)
	}

	// 备份并替换二进制：写 .new 后原子 rename（Linux 允许替换运行中的可执行文件）
	if err := copyFile(exePath, filepath.Join(backupDir, filepath.Base(exePath)), 0o755); err != nil {
		return nil, fmt.Errorf("备份当前程序失败: %w", err)
	}
	tmpBin := exePath + ".new"
	if err := copyFile(newBin, tmpBin, 0o755); err != nil {
		return nil, fmt.Errorf("写入新程序失败: %w", err)
	}
	if err := os.Rename(tmpBin, exePath); err != nil {
		_ = os.Remove(tmpBin)
		return nil, fmt.Errorf("替换程序失败: %w", err)
	}

	// 替换 public/（如更新包携带）
	newPublic := filepath.Join(extractDir, "public")
	if fi, err := os.Stat(newPublic); err == nil && fi.IsDir() {
		oldPublic := filepath.Join(root, "public")
		if _, err := os.Stat(oldPublic); err == nil {
			if err := os.Rename(oldPublic, filepath.Join(backupDir, "public")); err != nil {
				return nil, fmt.Errorf("备份 public 目录失败: %w", err)
			}
		}
		if err := os.Rename(newPublic, oldPublic); err != nil {
			return nil, fmt.Errorf("替换 public 目录失败: %w", err)
		}
	}

	// 合并 migrations/（如更新包携带；重启后 RunMigrations 自动执行增量）
	newMigrations := filepath.Join(extractDir, "migrations")
	if fi, err := os.Stat(newMigrations); err == nil && fi.IsDir() {
		if err := copyTree(newMigrations, filepath.Join(root, "migrations")); err != nil {
			return nil, fmt.Errorf("合并 migrations 目录失败: %w", err)
		}
	}

	_ = os.RemoveAll(extractDir)
	slog.Info("更新包应用完成", "from", version.Version, "to", newVersion, "backup", backupDir)
	return &UpgradeResult{
		Version:    newVersion,
		Applied:    true,
		Restarting: true,
		BackupDir:  backupDir,
		Message:    fmt.Sprintf("已升级到 %s，服务即将自动重启", newVersion),
	}, nil
}

// restartProcess 以相同参数自替换进程镜像；失败则退出交由 systemd 等进程管理器拉起
func restartProcess() {
	exe, err := os.Executable()
	if err == nil {
		slog.Info("升级完成，正在重启进程", "exe", exe)
		if err := syscall.Exec(exe, os.Args, os.Environ()); err != nil {
			slog.Error("进程自重启失败，将直接退出（请由进程管理器拉起）", "err", err)
		}
	}
	os.Exit(0)
}
