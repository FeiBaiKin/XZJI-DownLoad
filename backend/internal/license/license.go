// Package license 实现「授权升级中心」（https://www.xzji.top）的客户端接入：
// 激活、心跳、Ed25519 响应验签、本地缓存与宽限期状态机、升级下载与应用。
// 协议契约见授权中心《接入文档》，客户端强制规则：
//  1. 验签失败视为无效响应，按断网处理（进入宽限期），绝不采信；
//  2. 响应 timestamp 与本地时间偏差超过 10 分钟拒绝（防重放）；
//  3. 验签通过且 status != active 按到期/吊销处理（停止对外服务）；
//  4. 心跳成功后整个响应缓存到本地，服务器不可达时凭缓存运行，
//     缓存 timestamp 距今超过 grace_period_days 天后停止对外服务。
package license

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"xzji/backend/internal/util"
	"xzji/backend/internal/version"
)

// 授权中心接入常量。公钥必须硬编码在程序内（协议要求），
// 用于验证授权响应签名，防止假授权服务器伪造响应。
const (
	CenterBaseURL = "https://www.xzji.top"
	ProductSlug   = "xzji"
	PublicKeyB64  = "JtMYJerOlRTZ/SOxEM5Jt4TOPxhvF3WGuaqfxnWlFdg="
)

// 授权状态值（与中心契约一致）
const (
	StatusActive         = "active"
	StatusExpired        = "expired"
	StatusRevoked        = "revoked"
	StatusUnactivated    = "unactivated"
	StatusDomainMismatch = "domain_mismatch"

	// ReasonGraceExceeded 本地判定：缓存超过宽限期（并非中心下发的状态）
	ReasonGraceExceeded = "grace_exceeded"
)

// BlockedMessage 停止对外服务时的统一对外文案（接入文档约定）
const BlockedMessage = "授权已过期，请联系服务商"

// UpdateInfo 可用更新信息（签名响应的附带字段，本身不参与签名，
// 版本号与哈希以签名字段 latest_version / sha256 为准）
type UpdateInfo struct {
	Version   string `json:"version"`
	Changelog string `json:"changelog"`
	FileSize  int64  `json:"file_size"`
	SHA256    string `json:"sha256"`
	Forced    bool   `json:"forced"`
}

// StatusResponse 中心签名状态响应（激活/心跳同构）
type StatusResponse struct {
	Product                string      `json:"product"`
	Code                   string      `json:"code"`
	Domain                 string      `json:"domain"`
	Status                 string      `json:"status"`
	ExpiresAt              int64       `json:"expires_at"` // unix 秒，0 = 永久
	Timestamp              int64       `json:"timestamp"`
	LatestVersion          string      `json:"latest_version"`
	SHA256                 string      `json:"sha256"`
	HeartbeatIntervalHours int         `json:"heartbeat_interval_hours"`
	GracePeriodDays        int         `json:"grace_period_days"`
	Signature              string      `json:"signature"`
	Update                 *UpdateInfo `json:"update,omitempty"`
	Message                string      `json:"message,omitempty"`
}

// verifySignature 按协议顺序拼接签名字段并用硬编码公钥验签
func verifySignature(r *StatusResponse) bool {
	pub, err := base64.StdEncoding.DecodeString(PublicKeyB64)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return false
	}
	sig, err := base64.StdEncoding.DecodeString(r.Signature)
	if err != nil {
		return false
	}
	msg := strings.Join([]string{
		r.Product, r.Code, r.Domain, r.Status,
		strconv.FormatInt(r.ExpiresAt, 10),
		strconv.FormatInt(r.Timestamp, 10),
		r.LatestVersion, r.SHA256,
		strconv.Itoa(r.HeartbeatIntervalHours),
		strconv.Itoa(r.GracePeriodDays),
	}, "\n")
	return ed25519.Verify(ed25519.PublicKey(pub), []byte(msg), sig)
}

// NormalizeDomain 与中心 util.NormalizeDomain 同款归一化：
// 小写、剥离协议/路径/端口，保证两端对同一 siteURL 得到相同域名串
func NormalizeDomain(input string) string {
	s := strings.TrimSpace(strings.ToLower(input))
	if s == "" {
		return ""
	}
	if strings.Contains(s, "://") {
		if u, err := url.Parse(s); err == nil && u.Host != "" {
			s = u.Host
		}
	}
	if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}
	if i := strings.LastIndex(s, ":"); i >= 0 && !strings.Contains(s, "]") {
		if _, err := strconv.Atoi(s[i+1:]); err == nil {
			s = s[:i]
		}
	}
	return s
}

// Manager 授权客户端：持有最近一次验签通过的状态响应，
// 对外提供 Valid() 判定与激活/心跳/升级操作。并发安全。
type Manager struct {
	domain    string
	cachePath string
	client    *http.Client

	mu          sync.RWMutex // 保护 resp / lastErr / lastCheckAt
	resp        *StatusResponse
	lastErr     string
	lastCheckAt time.Time

	opMu sync.Mutex // 序列化激活/心跳网络操作
	upMu sync.Mutex // 升级互斥（TryLock 防重复触发）

	stopCh   chan struct{}
	stopOnce sync.Once
}

// NewManager 创建授权客户端。siteURL 取自 config.yaml 的 allowed_origins[0]
// （安装向导写入的站点地址），归一化后作为授权绑定域名。
func NewManager(siteURL string) *Manager {
	domain := NormalizeDomain(siteURL)
	if domain == "" {
		domain = "localhost" // 未配置站点地址时的兜底（本地开发）
	}
	return &Manager{
		domain:    domain,
		cachePath: filepath.Join(util.BackendRoot(), "license.json"),
		client:    &http.Client{Timeout: 20 * time.Second},
		stopCh:    make(chan struct{}),
	}
}

// Domain 当前站点的授权绑定域名
func (m *Manager) Domain() string { return m.domain }

// Load 读取本地缓存的签名授权响应。验签失败视为无凭证（文件被篡改或损坏）。
func (m *Manager) Load() {
	data, err := os.ReadFile(m.cachePath)
	if err != nil {
		return // 无缓存 = 未激活
	}
	var r StatusResponse
	if err := json.Unmarshal(data, &r); err != nil {
		slog.Warn("授权缓存解析失败，忽略", "err", err)
		return
	}
	if !verifySignature(&r) {
		slog.Warn("授权缓存验签失败，忽略（文件可能被篡改）", "path", m.cachePath)
		return
	}
	m.mu.Lock()
	m.resp = &r
	m.mu.Unlock()
	valid, reason := m.evaluate(&r, time.Now())
	slog.Info("已加载本地授权缓存", "status", r.Status, "domain", r.Domain, "valid", valid, "reason", reason)
}

// evaluate 按客户端强制规则判定授权可用性，返回 (是否有效, 无效原因)
func (m *Manager) evaluate(r *StatusResponse, now time.Time) (bool, string) {
	if r == nil {
		return false, StatusUnactivated
	}
	// 缓存与当前站点/产品不匹配：防止 license.json 被拷贝到其他部署
	if r.Product != ProductSlug || r.Domain != m.domain {
		return false, StatusDomainMismatch
	}
	if r.Status != StatusActive {
		return false, r.Status
	}
	if r.ExpiresAt != 0 && now.Unix() > r.ExpiresAt {
		return false, StatusExpired
	}
	if now.Unix() > m.graceDeadline(r) {
		return false, ReasonGraceExceeded
	}
	return true, ""
}

// graceDeadline 宽限期截止：最近一次验签通过响应的 timestamp + 宽限天数。
// 下限 1 天，避免中心误配 0 导致两次心跳之间被判停服。
func (m *Manager) graceDeadline(r *StatusResponse) int64 {
	days := r.GracePeriodDays
	if days < 1 {
		days = 1
	}
	return r.Timestamp + int64(days)*86400
}

// Valid 授权当前是否有效（LicenseGuard 中间件每请求调用，仅做内存判定）
func (m *Manager) Valid() bool {
	m.mu.RLock()
	r := m.resp
	m.mu.RUnlock()
	ok, _ := m.evaluate(r, time.Now())
	return ok
}

// callCenter 调用中心签名状态接口（activate/heartbeat），完成验签与时间戳偏差校验。
// 返回 error 时（网络/验签/重放）调用方一律按「本次联系失败」处理，不得改动现有缓存。
func (m *Manager) callCenter(path, code string) (*StatusResponse, error) {
	body, _ := json.Marshal(map[string]string{
		"product": ProductSlug,
		"code":    code,
		"domain":  m.domain,
		"version": version.Version,
	})
	req, err := http.NewRequest(http.MethodPost, CenterBaseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("无法连接授权中心: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("读取授权中心响应失败: %w", err)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, errors.New("请求过于频繁，请稍后重试")
	}
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &e)
		if e.Error != "" {
			return nil, errors.New(e.Error)
		}
		return nil, fmt.Errorf("授权中心返回异常状态 %d", resp.StatusCode)
	}
	var sr StatusResponse
	if err := json.Unmarshal(raw, &sr); err != nil {
		return nil, fmt.Errorf("解析授权响应失败: %w", err)
	}
	if !verifySignature(&sr) {
		return nil, errors.New("授权响应验签失败（可能存在伪造的授权服务器），已拒绝")
	}
	if d := time.Since(time.Unix(sr.Timestamp, 0)); d > 10*time.Minute || d < -10*time.Minute {
		return nil, errors.New("授权响应时间戳偏差超过 10 分钟，已拒绝（请检查服务器系统时间）")
	}
	return &sr, nil
}

// adopt 采信一份验签通过的响应：更新内存状态并落盘缓存
func (m *Manager) adopt(r *StatusResponse) {
	m.mu.Lock()
	m.resp = r
	m.lastErr = ""
	m.lastCheckAt = time.Now()
	m.mu.Unlock()
	data, _ := json.MarshalIndent(r, "", "  ")
	if err := os.WriteFile(m.cachePath, data, 0o600); err != nil {
		slog.Warn("授权缓存写入失败", "err", err, "path", m.cachePath)
	}
}

func (m *Manager) noteFailure(err error) {
	m.mu.Lock()
	m.lastErr = err.Error()
	m.lastCheckAt = time.Now()
	m.mu.Unlock()
}

// Activate 用授权码向中心激活本站点（同域名重复激活幂等，适用于重装）。
// 缓存覆盖策略：同一授权码总是采信最新响应；更换授权码时仅当新响应为
// active 才覆盖，防止误输入绑定了其他域名的码顶掉现有有效授权。
//
// domain 为本次激活探测到的本站域名（归一化后）；非空时覆盖 m.domain，
// 使本次及后续心跳/校验均使用新域名；为空时沿用配置中已有的 m.domain。
func (m *Manager) Activate(code, domain string) (*StatusResponse, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, errors.New("请输入授权码")
	}
	if domain != "" {
		m.mu.Lock()
		m.domain = domain
		m.mu.Unlock()
	}
	m.opMu.Lock()
	defer m.opMu.Unlock()
	sr, err := m.callCenter("/api/v1/activate", code)
	if err != nil {
		m.noteFailure(err)
		return nil, err
	}
	m.mu.RLock()
	old := m.resp
	m.mu.RUnlock()
	if old != nil && old.Code != sr.Code && sr.Status != StatusActive {
		return sr, nil // 新码未激活成功，保留现有缓存，结果仅供管理员查看
	}
	m.adopt(sr)
	return sr, nil
}

// Heartbeat 向中心心跳一次并刷新缓存。
// 网络/验签失败按断网处理：保留现有缓存（进入宽限期），返回错误。
func (m *Manager) Heartbeat() (*StatusResponse, error) {
	m.mu.RLock()
	cached := m.resp
	m.mu.RUnlock()
	if cached == nil {
		return nil, errors.New("尚未激活，请先输入授权码")
	}
	m.opMu.Lock()
	defer m.opMu.Unlock()
	sr, err := m.callCenter("/api/v1/heartbeat", cached.Code)
	if err != nil {
		m.noteFailure(err)
		return nil, err
	}
	m.adopt(sr)
	return sr, nil
}

// Start 启动心跳循环：启动 5s 后首跳（程序启动时应心跳一次，见接入文档），
// 之后按响应的 heartbeat_interval_hours 定时；失败 15 分钟后重试；
// 未激活时每 30 分钟复查（等待管理员在后台激活）。
func (m *Manager) Start() {
	go func() {
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		for {
			select {
			case <-m.stopCh:
				return
			case <-timer.C:
			}
			timer.Reset(m.tick())
		}
	}()
	slog.Info("授权心跳循环已启动", "center", CenterBaseURL, "domain", m.domain, "version", version.Version)
}

// tick 执行一次心跳，返回下次执行间隔
func (m *Manager) tick() (next time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("授权心跳异常", "err", r)
			next = 15 * time.Minute
		}
	}()
	m.mu.RLock()
	activated := m.resp != nil
	m.mu.RUnlock()
	if !activated {
		return 30 * time.Minute
	}
	sr, err := m.Heartbeat()
	if err != nil {
		slog.Warn("授权心跳失败，将凭本地缓存运行（宽限期内）", "err", err)
		return 15 * time.Minute
	}
	hours := sr.HeartbeatIntervalHours
	if hours < 1 {
		hours = 6
	}
	valid, reason := m.evaluate(sr, time.Now())
	slog.Info("授权心跳成功", "status", sr.Status, "valid", valid, "reason", reason, "nextIntervalHours", hours)
	return time.Duration(hours) * time.Hour
}

// Stop 停止心跳循环（优雅关闭时调用）
func (m *Manager) Stop() {
	m.stopOnce.Do(func() { close(m.stopCh) })
}

// Snapshot 授权状态快照（管理后台展示用）
type Snapshot struct {
	Activated              bool        `json:"activated"`
	Valid                  bool        `json:"valid"`
	Reason                 string      `json:"reason,omitempty"`
	Status                 string      `json:"status,omitempty"`
	Code                   string      `json:"code,omitempty"`
	Domain                 string      `json:"domain"`
	ExpiresAt              int64       `json:"expiresAt"` // 0 = 永久，不可 omitempty
	LastVerifiedAt         int64       `json:"lastVerifiedAt,omitempty"`
	GraceDeadline          int64       `json:"graceDeadline,omitempty"`
	HeartbeatIntervalHours int         `json:"heartbeatIntervalHours,omitempty"`
	GracePeriodDays        int         `json:"gracePeriodDays,omitempty"`
	CurrentVersion         string      `json:"currentVersion"`
	LatestVersion          string      `json:"latestVersion,omitempty"`
	Update                 *UpdateInfo `json:"update,omitempty"`
	LastError              string      `json:"lastError,omitempty"`
	LastCheckAt            int64       `json:"lastCheckAt,omitempty"`
}

// Snapshot 生成当前授权状态快照
func (m *Manager) Snapshot() Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s := Snapshot{
		Domain:         m.domain,
		CurrentVersion: version.Version,
		LastError:      m.lastErr,
	}
	if !m.lastCheckAt.IsZero() {
		s.LastCheckAt = m.lastCheckAt.Unix()
	}
	r := m.resp
	s.Valid, s.Reason = m.evaluate(r, time.Now())
	if r == nil {
		return s
	}
	s.Activated = true
	s.Status = r.Status
	s.Code = r.Code
	s.ExpiresAt = r.ExpiresAt
	s.LastVerifiedAt = r.Timestamp
	s.GraceDeadline = m.graceDeadline(r)
	s.HeartbeatIntervalHours = r.HeartbeatIntervalHours
	s.GracePeriodDays = r.GracePeriodDays
	s.LatestVersion = r.LatestVersion
	s.Update = r.Update
	return s
}
