package license

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"xzji/backend/internal/util"
)

// UpgradeResult 升级操作结果
type UpgradeResult struct {
	Version     string `json:"version"`
	Applied     bool   `json:"applied"`    // 是否已替换文件（false = 仅下载暂存，需手动替换）
	Restarting  bool   `json:"restarting"` // 是否即将自动重启进程
	StagingPath string `json:"stagingPath,omitempty"`
	BackupDir   string `json:"backupDir,omitempty"`
	Message     string `json:"message"`
}

// upgradeToken 中心 /api/v1/upgrade/request 响应（一次性下载令牌，15 分钟有效）
type upgradeToken struct {
	URL       string `json:"url"`
	Version   string `json:"version"`
	Changelog string `json:"changelog"`
	FileSize  int64  `json:"file_size"`
	SHA256    string `json:"sha256"`
	Forced    bool   `json:"forced"`
	ExpiresAt int64  `json:"expires_at"`
}

// Upgrade 执行完整升级：心跳刷新签名状态 → 请求下载令牌 → 下载并校验 SHA256 →
// 应用更新包（平台相关，见 apply_*.go）。
// 哈希与签名响应中 sha256 不一致时一律丢弃（防升级通道劫持，接入文档第 130 行规则）。
//
// 更新包约定：tar.gz，根目录含新版二进制（xzji-linux-amd64 或与当前可执行文件同名），
// 可选 public/（前端构建产物）与 migrations/（增量 SQL）；
// config.yaml / uploads/ / license.json / .installed 永不覆盖。
func (m *Manager) Upgrade() (*UpgradeResult, error) {
	if !m.upMu.TryLock() {
		return nil, errors.New("升级正在进行中，请勿重复操作")
	}
	defer m.upMu.Unlock()

	// 1. 先心跳：拿到最新的带签名版本号与更新包哈希（升级校验的信任锚点）
	sr, err := m.Heartbeat()
	if err != nil {
		return nil, fmt.Errorf("升级前心跳失败: %w", err)
	}
	if ok, reason := m.evaluate(sr, time.Now()); !ok {
		return nil, fmt.Errorf("授权无效（%s），无法升级", reason)
	}
	if sr.Update == nil {
		return nil, errors.New("当前已是最新版本，无可用更新")
	}
	signedSHA := strings.ToLower(sr.SHA256)
	if signedSHA == "" {
		return nil, errors.New("签名响应缺少更新包哈希，已中止")
	}

	// 2. 请求一次性下载令牌，并交叉校验其哈希与签名响应一致
	tok, err := m.requestUpgradeToken(sr.Code)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(tok.SHA256, signedSHA) {
		return nil, errors.New("下载令牌的文件哈希与签名响应不一致，已中止（可能存在升级通道劫持）")
	}

	// 3. 下载到暂存目录并校验 SHA256
	stagingDir := filepath.Join(util.BackendRoot(), "update_staging")
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建暂存目录失败: %w", err)
	}
	pkgPath := filepath.Join(stagingDir, fmt.Sprintf("%s-%s.tar.gz", ProductSlug, tok.Version))
	if err := m.downloadTo(tok.URL, pkgPath, signedSHA, tok.FileSize); err != nil {
		return nil, err
	}
	slog.Info("更新包下载并校验完成", "version", tok.Version, "path", pkgPath, "size", tok.FileSize)

	// 4. 应用更新包（Linux 生产环境替换文件并重启；其余平台仅暂存）
	return m.apply(pkgPath, tok.Version)
}

// requestUpgradeToken 请求一次性下载令牌（授权须在有效期内且域名匹配，中心侧校验）
func (m *Manager) requestUpgradeToken(code string) (*upgradeToken, error) {
	body, _ := json.Marshal(map[string]string{
		"product": ProductSlug,
		"code":    code,
		"domain":  m.domain,
	})
	resp, err := m.client.Post(CenterBaseURL+"/api/v1/upgrade/request", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("请求下载令牌失败: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("读取下载令牌响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &e)
		if e.Error != "" {
			return nil, errors.New(e.Error)
		}
		return nil, fmt.Errorf("请求下载令牌失败（HTTP %d）", resp.StatusCode)
	}
	var t upgradeToken
	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, fmt.Errorf("解析下载令牌失败: %w", err)
	}
	if t.URL == "" {
		return nil, errors.New("下载令牌响应缺少下载地址")
	}
	return &t, nil
}

// downloadTo 流式下载更新包到 dest 并校验 SHA256，不一致则删除文件并报错
func (m *Manager) downloadTo(urlPath, dest, wantSHA string, expectSize int64) error {
	dl := &http.Client{Timeout: 10 * time.Minute}
	resp, err := dl.Get(CenterBaseURL + urlPath)
	if err != nil {
		return fmt.Errorf("下载更新包失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &e)
		if e.Error != "" {
			return fmt.Errorf("下载更新包失败: %s", e.Error)
		}
		return fmt.Errorf("下载更新包失败（HTTP %d）", resp.StatusCode)
	}

	maxBytes := expectSize
	if maxBytes <= 0 {
		maxBytes = 2 << 30 // 中心未给出大小时的上限保护
	}
	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("写入更新包失败: %w", err)
	}
	hasher := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(f, hasher), io.LimitReader(resp.Body, maxBytes))
	closeErr := f.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(dest)
		return fmt.Errorf("保存更新包失败: %v", errors.Join(copyErr, closeErr))
	}
	got := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(got, wantSHA) {
		_ = os.Remove(dest)
		return errors.New("更新包 SHA256 校验失败，已丢弃（可能存在升级通道劫持）")
	}
	return nil
}

// extractTarGz 安全解包 tar.gz 到 destDir：拒绝绝对路径与 .. 穿越，跳过符号链接
func extractTarGz(src, destDir string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("更新包不是有效的 tar.gz: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("读取更新包内容失败: %w", err)
		}
		name := filepath.Clean(hdr.Name)
		if filepath.IsAbs(name) || strings.HasPrefix(name, "..") || strings.Contains(name, ".."+string(filepath.Separator)) {
			return fmt.Errorf("更新包含非法路径 %q，已中止", hdr.Name)
		}
		target := filepath.Join(destDir, name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(hdr.Mode) & 0o777
			if mode == 0 {
				mode = 0o644
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, tr) //nolint:gosec // 包整体大小已受下载上限约束
			closeErr := out.Close()
			if copyErr != nil || closeErr != nil {
				return fmt.Errorf("解包文件 %s 失败: %v", name, errors.Join(copyErr, closeErr))
			}
		default:
			// 符号链接等一律跳过：更新包约定只含普通文件与目录
			slog.Warn("更新包内跳过非常规条目", "name", hdr.Name, "type", hdr.Typeflag)
		}
	}
}

// findNewBinary 在解包目录根部定位新版二进制：
// 优先与当前可执行文件同名，其次约定名 xzji-linux-amd64，最后唯一的可执行普通文件
func findNewBinary(extractDir, exeBase string) string {
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return ""
	}
	var execCandidates []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if e.Name() == exeBase {
			return filepath.Join(extractDir, e.Name())
		}
		if info, err := e.Info(); err == nil && info.Mode()&0o111 != 0 {
			execCandidates = append(execCandidates, e.Name())
		}
	}
	for _, e := range entries {
		if !e.IsDir() && e.Name() == "xzji-linux-amd64" {
			return filepath.Join(extractDir, e.Name())
		}
	}
	if len(execCandidates) == 1 {
		return filepath.Join(extractDir, execCandidates[0])
	}
	return ""
}

// copyFile 复制普通文件（保留权限位）
func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	return errors.Join(copyErr, closeErr)
}

// copyTree 递归叠加复制目录（只增改不删除，用于 migrations/ 合并）
func copyTree(srcDir, dstDir string) error {
	return filepath.Walk(srcDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dstDir, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		return copyFile(p, target, info.Mode()&0o777)
	})
}

// ScheduleRestart 延迟重启进程（先让 HTTP 响应发出再重启，平台实现见 restart_*.go）
func (m *Manager) ScheduleRestart(delay time.Duration) {
	go func() {
		time.Sleep(delay)
		m.Stop()
		restartProcess()
	}()
}
