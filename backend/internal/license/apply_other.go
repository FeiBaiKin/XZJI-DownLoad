//go:build !linux

package license

import (
	"fmt"
	"log/slog"
)

// apply 非 Linux 平台（本地开发等）：仅下载暂存并通过校验，不做自动替换。
// 生产部署形态为 Linux 单二进制，自动替换与重启见 apply_linux.go。
func (m *Manager) apply(pkgPath, newVersion string) (*UpgradeResult, error) {
	return &UpgradeResult{
		Version:     newVersion,
		Applied:     false,
		StagingPath: pkgPath,
		Message:     fmt.Sprintf("更新包 %s 已下载并通过校验，当前平台不支持自动替换，请手动部署", newVersion),
	}, nil
}

func restartProcess() {
	slog.Warn("当前平台不支持自动重启，请手动重启服务")
}
