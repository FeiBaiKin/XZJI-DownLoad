package util

import (
	"os"
	"path/filepath"
)

// BackendRoot 返回 backend 目录的绝对路径。
// 优先使用工作目录（兼容 go run 调试），其次用可执行文件目录（生产环境）。
// 原在 router/handler/service 三处各有重复实现，现已收敛至此。
func BackendRoot() string {
	wd, err := os.Getwd()
	if err == nil {
		dir := wd
		for i := 0; i < 3; i++ {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	if exe, err := os.Executable(); err == nil {
		return filepath.Dir(exe)
	}
	return "."
}
