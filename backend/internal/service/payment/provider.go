package payment

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"xzji/backend/internal/models"
)

// PaymentHTTPClient 所有支付网关请求共享的 HTTP 客户端。
// 设置合理超时避免网关慢响应导致 goroutine 堆积（http.DefaultClient 无超时）。
var PaymentHTTPClient = &http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConns:          50,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

// ====== Provider 接口 ======

type Order struct {
	OrderNo string
	Amount  float64
	Points  int // 赠送经验
	UserID  uint
	Subject string // 网关侧商品标题，空则回退「账户充值」
}

// subjectOr 取网关侧商品标题：未显式指定时回退充值文案（保持历史订单展示不变）
func (o Order) subjectOr(fallback string) string {
	if strings.TrimSpace(o.Subject) != "" {
		return o.Subject
	}
	return fallback
}

type PaymentConfig struct {
	SiteURL string
	// 微信
	WxAppId, WxMchId, WxSerialNo, WxApiV3Key, WxPrivateKey, WxNotifyUrl string
	// 支付宝
	AliAppId, AliPrivateKey, AliPublicKey, AliScene, AliNotifyUrl, AliReturnUrl string
	// 易支付
	EpayGateway, EpayPid, EpayKey, EpayNotifyUrl, EpayReturnUrl string
	// 已启用的支付方式（来自 site_settings.payment_enabled_providers）
	EnabledProviders []string
}

type PaymentResult struct {
	Type   string `json:"type"`   // wechat / alipay / epay
	Scene  string `json:"scene"`  // native / page / precreate / v1
	PayUrl string `json:"payUrl"` // 支付链接或二维码内容
	QRCode string `json:"qrCode"` // 二维码数据（微信 native 返回 code_url）
}

type CallbackRequest struct {
	Method  string
	Body    []byte
	Query   map[string]string
	Headers map[string]string
}

type CallbackResult struct {
	Verified bool
	Skip     bool // 跳过处理（资源事件等）
	Error    string
	OrderNo  string
	Amount   float64
	TradeNo  string
}

type Provider interface {
	Name() string
	Label() string
	Scenes() []string
	CreatePayment(order Order, cfg PaymentConfig, scene string, extra map[string]string) (PaymentResult, error)
	VerifyCallback(req CallbackRequest, cfg PaymentConfig) (CallbackResult, error)
}

// ====== 注册表 ======

var providers = map[string]Provider{}

func Register(p Provider) { providers[p.Name()] = p }
func Get(name string) (Provider, error) {
	if p, ok := providers[name]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("不支持的支付方式: %s", name)
}
func ListMethods() []map[string]any {
	var out []map[string]any
	for _, p := range providers {
		out = append(out, map[string]any{"name": p.Name(), "label": p.Label(), "scenes": p.Scenes()})
	}
	return out
}

// ====== 通用：获取支付配置 ======

func GetPaymentConfig(db *gorm.DB) PaymentConfig {
	var rows []models.SiteSetting
	db.Where("setting_key LIKE ?", "payment_%").Find(&rows)
	cfg := PaymentConfig{}

	m := map[string]string{}
	for _, r := range rows {
		m[r.SettingKey] = r.SettingValue
	}

	siteURL := strings.TrimRight(m["payment_site_url"], "/")
	cfg.SiteURL = siteURL

	// 已启用的支付方式（payment_enabled_providers 存为 JSON 字符串）
	if raw := m["payment_enabled_providers"]; raw != "" {
		var arr []string
		if err := json.Unmarshal([]byte(raw), &arr); err == nil && len(arr) > 0 {
			cfg.EnabledProviders = arr
		}
	}
	if cfg.EnabledProviders == nil {
		cfg.EnabledProviders = []string{}
	}

	cfg.WxAppId = m["payment_wx_appid"]
	cfg.WxMchId = m["payment_wx_mchid"]
	cfg.WxSerialNo = m["payment_wx_serial_no"]
	cfg.WxApiV3Key = m["payment_wx_api_v3_key"]
	cfg.WxPrivateKey = m["payment_wx_private_key"]
	cfg.WxNotifyUrl = normalizeURL(m["payment_wx_notify_url"])
	if cfg.WxNotifyUrl == "" && siteURL != "" {
		cfg.WxNotifyUrl = siteURL + "/api/payment/callback/wechat"
	}

	cfg.AliAppId = m["payment_ali_appid"]
	cfg.AliPrivateKey = m["payment_ali_private_key"]
	cfg.AliPublicKey = m["payment_ali_public_key"]
	cfg.AliScene = m["payment_ali_scene"]
	if cfg.AliScene == "" {
		cfg.AliScene = "page"
	}
	cfg.AliNotifyUrl = normalizeURL(m["payment_ali_notify_url"])
	if cfg.AliNotifyUrl == "" && siteURL != "" {
		cfg.AliNotifyUrl = siteURL + "/api/payment/callback/alipay"
	}
	cfg.AliReturnUrl = normalizeURL(m["payment_ali_return_url"])
	if cfg.AliReturnUrl == "" && siteURL != "" {
		cfg.AliReturnUrl = siteURL + "/?from=alipay"
	}

	cfg.EpayGateway = m["payment_epay_gateway"]
	cfg.EpayPid = m["payment_epay_pid"]
	cfg.EpayKey = m["payment_epay_key"]
	cfg.EpayNotifyUrl = normalizeURL(m["payment_epay_notify_url"])
	if cfg.EpayNotifyUrl == "" && siteURL != "" {
		cfg.EpayNotifyUrl = siteURL + "/api/payment/callback/epay"
	}
	cfg.EpayReturnUrl = normalizeURL(m["payment_epay_return_url"])
	if cfg.EpayReturnUrl == "" && siteURL != "" {
		cfg.EpayReturnUrl = siteURL + "/?from=epay"
	}

	return cfg
}

// normalizeURL 清理 URL 中路径部分的连续斜杠（如 //api → /api），保留协议双斜杠 (https://)
func normalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// 分离协议和路径：https://domain.com//path → scheme=https, rest=//domain.com//path
	var scheme, rest string
	if idx := strings.Index(raw, "://"); idx >= 0 {
		scheme = raw[:idx+3] // "https://"
		rest = raw[idx+3:]   // "domain.com//path"
	} else {
		rest = raw
	}
	// 将路径中的连续斜杠合并为单个
	for strings.Contains(rest, "//") {
		rest = strings.ReplaceAll(rest, "//", "/")
	}
	// 去除末尾斜杠
	rest = strings.TrimRight(rest, "/")
	return scheme + rest
}

// ====== 通用：processPaidOrder（事务，对齐原项目） ======

func ProcessPaidOrder(db *gorm.DB, orderNo string, paidAmount, tradeNo string) (uint, map[string]any, error) {
	tx := db.Begin()
	if tx.Error != nil {
		return 0, nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var order models.PaymentOrder
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("order_no = ?", orderNo).First(&order).Error; err != nil {
		tx.Rollback()
		return 0, nil, fmt.Errorf("订单不存在")
	}

	// 金额前置校验
	amt := 0.0
	fmt.Sscanf(paidAmount, "%f", &amt)
	if amt <= 0 {
		tx.Rollback()
		return 0, nil, fmt.Errorf("无效的支付金额")
	}
	if diff := amt - order.Amount; diff > 0.01 || diff < -0.01 {
		tx.Rollback()
		return 0, nil, fmt.Errorf("支付金额不匹配")
	}
	if order.Status == "paid" {
		if err := tx.Commit().Error; err != nil {
			return 0, nil, err
		}
		return order.UserID, map[string]any{"success": true, "alreadyPaid": true}, nil
	}

	// 更新订单
	now := time.Now()
	if err := tx.Model(&order).Updates(map[string]any{"status": "paid", "paid_at": now, "trade_no": tradeNo}).Error; err != nil {
		tx.Rollback()
		return 0, nil, err
	}

	if order.OrderType == models.OrderTypeMembership {
		// 会员直付：不入余额，直接开通/续期（省去「先充值再扣余额」）
		if err := activateMembership(tx, &order, amt, now); err != nil {
			tx.Rollback()
			return 0, nil, err
		}
	} else {
		// 加余额（使用回调金额，与原项目一致）
		if err := tx.Model(&models.User{}).Where("id = ?", order.UserID).Update("balance", gorm.Expr("balance + ?", amt)).Error; err != nil {
			tx.Rollback()
			return 0, nil, err
		}
		// 流水
		if err := tx.Create(&models.PointsTransaction{
			UserID: order.UserID, Type: "recharge", Amount: amt, BalanceAmount: amt,
			ExpEarned: order.Points, Description: "账户充值", RefID: &order.ID,
		}).Error; err != nil {
			tx.Rollback()
			return 0, nil, err
		}
	}

	// 赠经验（两类订单同口径：直付会员与「充值同额再购买」拿到的经验一致）
	if order.Points > 0 {
		if err := tx.Model(&models.User{}).Where("id = ?", order.UserID).Update("exp", gorm.Expr("exp + ?", order.Points)).Error; err != nil {
			tx.Rollback()
			return 0, nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return 0, nil, err
	}
	return order.UserID, map[string]any{"success": true}, nil
}

// activateMembership 会员直付订单入账：按档位天数开通/续期，并补齐会员订单与流水。
// 到期时间口径与余额购买一致（未过期则从原到期日叠加）。
func activateMembership(tx *gorm.DB, order *models.PaymentOrder, amt float64, now time.Time) error {
	if order.MembershipLevel == "" {
		return fmt.Errorf("会员订单缺少档位信息")
	}
	// 行锁用户，避免并发回调把同一笔有效期算重
	var user models.User
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, order.UserID).Error; err != nil {
		return fmt.Errorf("用户不存在")
	}
	days := order.MembershipDays
	if days <= 0 {
		// 下单后管理员改过档位天数（或历史订单缺列）时按当前配置兜底
		var cfg models.MembershipConfig
		if err := tx.Where("level_name = ?", order.MembershipLevel).First(&cfg).Error; err != nil {
			return fmt.Errorf("会员档位不存在")
		}
		days = cfg.Days
	}
	base := now
	if user.MembershipExpiresAt != nil && user.MembershipExpiresAt.After(base) {
		base = *user.MembershipExpiresAt
	}
	expiresAt := base.AddDate(0, 0, days)
	if err := tx.Model(&models.User{}).Where("id = ?", order.UserID).Updates(map[string]any{
		"membership_level":      order.MembershipLevel,
		"membership_expires_at": expiresAt,
	}).Error; err != nil {
		return err
	}
	if err := tx.Create(&models.MembershipOrder{
		OrderNo: order.OrderNo, UserID: order.UserID, MembershipLevel: order.MembershipLevel,
		Amount: amt, Days: days, Status: "paid", PaidAt: &now,
	}).Error; err != nil {
		return err
	}
	// 余额未变动，Amount 记 0；实付金额写在描述里，避免流水里出现虚假的余额扣减
	return tx.Create(&models.PointsTransaction{
		UserID: order.UserID, Type: "membership", ExpEarned: order.Points,
		Description: fmt.Sprintf("开通%s（%s支付 %.2f 元）", order.MembershipLevel, methodLabel(order.PaymentMethod), amt),
		RefID:       &order.ID,
	}).Error
}

// methodLabel 支付方式中文名（流水描述用），未知方式原样返回
func methodLabel(name string) string {
	switch name {
	case "wechat":
		return "微信"
	case "alipay":
		return "支付宝"
	case "epay":
		return "易支付"
	}
	return name
}
