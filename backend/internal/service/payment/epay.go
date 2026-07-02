package payment

import (
	"crypto/md5"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

func init() { Register(EpayProvider{}) }

type EpayProvider struct{}

func (EpayProvider) Name() string     { return "epay" }
func (EpayProvider) Label() string    { return "易支付" }
func (EpayProvider) Scenes() []string { return []string{"v1"} }

func (EpayProvider) CreatePayment(order Order, cfg PaymentConfig, scene string, extra map[string]string) (PaymentResult, error) {
	if cfg.EpayGateway == "" || cfg.EpayPid == "" || cfg.EpayKey == "" {
		return PaymentResult{}, fmt.Errorf("易支付配置不完整")
	}
	amount := fmt.Sprintf("%.2f", order.Amount)
	params := map[string]string{
		"pid":          cfg.EpayPid,
		"out_trade_no": order.OrderNo,
		"notify_url":   cfg.EpayNotifyUrl,
		"return_url":   cfg.EpayReturnUrl,
		"name":         "账户充值 " + amount + "元",
		"money":        amount,
	}
	// 不指定 type：跳转彩虹统一收银台由用户自行选择支付通道（对齐原版）
	params["sign"] = epaySign(params, cfg.EpayKey)
	params["sign_type"] = "MD5"

	gateway := strings.TrimRight(cfg.EpayGateway, "/")
	payURL := gateway + "/submit.php?" + epayEncode(params)
	return PaymentResult{Type: "epay", Scene: "v1", PayUrl: payURL}, nil
}

func (EpayProvider) VerifyCallback(req CallbackRequest, cfg PaymentConfig) (CallbackResult, error) {
	if cfg.EpayKey == "" {
		return CallbackResult{Skip: true, Error: "易支付KEY未配置"}, nil
	}
	sig := req.Query["sign"]
	computed := epaySign(req.Query, cfg.EpayKey)
	if computed != sig {
		return CallbackResult{Verified: false, Error: "签名验证失败"}, nil
	}
	// 仅交易成功才入账（对齐原版 trade_status === 'TRADE_SUCCESS'）
	if req.Query["trade_status"] != "TRADE_SUCCESS" {
		return CallbackResult{Skip: true, Error: "订单状态: " + req.Query["trade_status"]}, nil
	}
	amount, _ := strconv.ParseFloat(req.Query["money"], 64)
	return CallbackResult{Verified: true, OrderNo: req.Query["out_trade_no"], Amount: amount, TradeNo: req.Query["trade_no"]}, nil
}

func epaySign(p map[string]string, key string) string {
	keys := make([]string, 0, len(p))
	for k := range p {
		if k == "sign" || k == "sign_type" || p[k] == "" { continue }
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys { parts = append(parts, k+"="+p[k]) }
	str := strings.Join(parts, "&") + key
	h := md5.Sum([]byte(str))
	return strings.ToLower(fmt.Sprintf("%x", h))
}

// epayEncode 构建查询串，对齐 JS encodeURIComponent（空格 -> %20）。
// 不能直接用 url.Values.Encode()：它把空格编码成 '+'，而部分易支付网关用
// rawurldecode 解析参数（'+' 不会还原为空格），导致回环值与签名串不一致，
// 网关回算的 MD5 与我方 sign 不符 → “MD5签名校验失败”。%20 在 urldecode 与
// rawurldecode 下都会还原为空格，可避免该问题。
func epayEncode(p map[string]string) string {
	v := url.Values{}
	for k, val := range p {
		v.Set(k, val)
	}
	return strings.ReplaceAll(v.Encode(), "+", "%20")
}
