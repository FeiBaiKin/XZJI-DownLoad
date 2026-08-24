package payment

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func init() { Register(AlipayProvider{}) }

type AlipayProvider struct{}

func (AlipayProvider) Name() string     { return "alipay" }
func (AlipayProvider) Label() string    { return "支付宝" }
func (AlipayProvider) Scenes() []string { return []string{"page", "precreate"} }

func (AlipayProvider) CreatePayment(order Order, cfg PaymentConfig, scene string, extra map[string]string) (PaymentResult, error) {
	if cfg.AliAppId == "" || cfg.AliPrivateKey == "" {
		return PaymentResult{}, fmt.Errorf("支付宝配置不完整")
	}
	amount := fmt.Sprintf("%.2f", order.Amount)
	if scene == "precreate" {
		return alipayPrecreate(order, cfg, amount)
	}
	// scene == "page"（跳转支付）：根据 UA 自动判断手机/电脑
	ua := ""
	if extra != nil {
		ua = extra["User-Agent"]
	}
	if isMobileUA(ua) {
		result, err := alipayWapPay(order, cfg, amount)
		if err != nil {
			slog.Error("[支付宝] 手机网站支付创建失败，回退到电脑网站支付", "err", err)
			return alipayPagePay(order, cfg, amount)
		}
		return result, nil
	}
	return alipayPagePay(order, cfg, amount)
}

// isMobileUA 根据 User-Agent 判断是否为移动浏览器
func isMobileUA(ua string) bool {
	if ua == "" {
		return false
	}
	ua = strings.ToLower(ua)
	mobileKeywords := []string{
		"mobile", "android", "iphone", "ipad", "ipod",
		"windows phone", "blackberry", "opera mini", "opera mobi",
		"webos", "symbian", "nokia", "samsung", "huawei",
		"xiaomi", "oppo", "vivo", "oneplus", "pixel",
	}
	for _, kw := range mobileKeywords {
		if strings.Contains(ua, kw) {
			return true
		}
	}
	return false
}

// alipayPagePay 电脑网站支付：返回可重定向到支付宝收银台的 URL
func alipayPagePay(order Order, cfg PaymentConfig, amount string) (PaymentResult, error) {
	biz := map[string]any{
		"out_trade_no": order.OrderNo,
		"total_amount": amount,
		"subject":      order.subjectOr("账户充值 " + amount + "元"),
		"product_code": "FAST_INSTANT_TRADE_PAY",
	}
	bizBytes, _ := json.Marshal(biz)
	params := map[string]string{
		"app_id":      cfg.AliAppId,
		"method":      "alipay.trade.page.pay",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"notify_url":  cfg.AliNotifyUrl,
		"return_url":  cfg.AliReturnUrl,
		"biz_content": string(bizBytes),
	}
	sig, err := signRSABase64(buildAliSignStr(params), cfg.AliPrivateKey)
	if err != nil {
		return PaymentResult{}, fmt.Errorf("签名失败: %v", err)
	}
	params["sign"] = sig
	payURL := "https://openapi.alipay.com/gateway.do?" + encodeParams(params)
	return PaymentResult{Type: "alipay", Scene: "page", PayUrl: payURL}, nil
}

// alipayWapPay 手机网站支付：返回可重定向到支付宝手机收银台的 URL
func alipayWapPay(order Order, cfg PaymentConfig, amount string) (PaymentResult, error) {
	biz := map[string]any{
		"out_trade_no": order.OrderNo,
		"total_amount": amount,
		"subject":      order.subjectOr("账户充值 " + amount + "元"),
		"product_code": "QUICK_WAP_WAY",
	}
	bizBytes, _ := json.Marshal(biz)
	params := map[string]string{
		"app_id":      cfg.AliAppId,
		"method":      "alipay.trade.wap.pay",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"notify_url":  cfg.AliNotifyUrl,
		"return_url":  cfg.AliReturnUrl,
		"biz_content": string(bizBytes),
	}
	sig, err := signRSABase64(buildAliSignStr(params), cfg.AliPrivateKey)
	if err != nil {
		return PaymentResult{}, fmt.Errorf("签名失败: %v", err)
	}
	params["sign"] = sig
	payURL := "https://openapi.alipay.com/gateway.do?" + encodeParams(params)
	return PaymentResult{Type: "alipay", Scene: "wap", PayUrl: payURL}, nil
}

// alipayPrecreate 当面付扫码：调用 alipay.trade.precreate，返回 qr_code 二维码链接
// （对齐原项目 _createPrecreate：POST 表单到网关 → 解析 JSON → 取 qr_code）
func alipayPrecreate(order Order, cfg PaymentConfig, amount string) (PaymentResult, error) {
	if cfg.AliNotifyUrl == "" {
		return PaymentResult{}, fmt.Errorf("支付宝当面付需要配置公网回调地址")
	}
	biz := map[string]any{
		"out_trade_no": order.OrderNo,
		"total_amount": amount,
		"subject":      order.subjectOr("账户充值 " + amount + "元"),
	}
	bizBytes, _ := json.Marshal(biz)
	params := map[string]string{
		"app_id":      cfg.AliAppId,
		"method":      "alipay.trade.precreate",
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"notify_url":  cfg.AliNotifyUrl,
		"biz_content": string(bizBytes),
	}
	sig, err := signRSABase64(buildAliSignStr(params), cfg.AliPrivateKey)
	if err != nil {
		return PaymentResult{}, fmt.Errorf("签名失败: %v", err)
	}
	params["sign"] = sig

	// POST 表单到支付宝网关。必须显式声明 charset=utf-8（对齐原项目 Node 的
	// Content-Type: application/x-www-form-urlencoded;charset=utf-8）：POST 表单时
	// 支付宝按 Content-Type 的字符集解读 body 字节，缺省 charset 会导致含中文的
	// biz_content 被误解析 → 验签串不一致 → "验签出错，请确认 charset…"。
	form := url.Values{}
	for k, v := range params {
		if v == "" {
			continue
		}
		form.Set(k, v)
	}
	req, err := http.NewRequest("POST", "https://openapi.alipay.com/gateway.do", strings.NewReader(form.Encode()))
	if err != nil {
		return PaymentResult{}, fmt.Errorf("构造请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	resp, err := PaymentHTTPClient.Do(req)
	if err != nil {
		return PaymentResult{}, fmt.Errorf("请求支付宝网关失败: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	// 支付宝部分错误响应使用 GBK 编码，直接按 UTF-8 解析会乱码；
	// 若非合法 UTF-8 则按 GBK 转码后再解析。
	if !utf8.Valid(body) {
		if decoded, err := simplifiedchinese.GBK.NewDecoder().Bytes(body); err == nil {
			body = decoded
		}
	}

	var result struct {
		Precreate struct {
			Code    string `json:"code"`
			Msg     string `json:"msg"`
			SubMsg  string `json:"sub_msg"`
			QrCode  string `json:"qr_code"`
		} `json:"alipay_trade_precreate_response"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return PaymentResult{}, fmt.Errorf("支付宝响应解析失败: %s", truncStr(string(body), 200))
	}
	if result.Precreate.Code != "10000" {
		msg := result.Precreate.SubMsg
		if msg == "" {
			msg = result.Precreate.Msg
		}
		if msg == "" {
			msg = "未知错误"
		}
		return PaymentResult{}, fmt.Errorf("支付宝当面付下单失败: %s", msg)
	}
	qr := result.Precreate.QrCode
	return PaymentResult{Type: "alipay", Scene: "precreate", PayUrl: qr, QRCode: qr}, nil
}

func truncStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func (AlipayProvider) VerifyCallback(req CallbackRequest, cfg PaymentConfig) (CallbackResult, error) {
	if cfg.AliPublicKey == "" {
		slog.Error("[支付宝] 支付宝公钥未配置，回调无法验签，请前往后台-支付配置填写支付宝公钥")
		return CallbackResult{Verified: false, Error: "支付宝公钥未配置"}, nil
	}
	params := req.Query
	if len(params) == 0 || params["sign"] == "" {
		slog.Error("[支付宝] 回调参数无效", "paramsLen", len(params))
		return CallbackResult{Verified: false, Error: "回调参数无效"}, nil
	}

	// 用支付宝公钥 RSA2 验签（签名串约定与下单侧 buildAliSignStr 一致：仅剔除 sign 与空值）
	if !verifyAliSign(params, cfg.AliPublicKey, params["sign"]) {
		slog.Error("[支付宝] 回调验签失败", "orderNo", params["out_trade_no"])
		return CallbackResult{Verified: false, Error: "签名验证失败"}, nil
	}

	// 仅 TRADE_SUCCESS 才入账，其余状态确认收到但跳过
	if params["trade_status"] != "TRADE_SUCCESS" {
		return CallbackResult{Skip: true, Error: "订单状态: " + params["trade_status"]}, nil
	}

	amount := 0.0
	fmt.Sscanf(params["total_amount"], "%f", &amount)
	return CallbackResult{
		Verified: true,
		OrderNo:  params["out_trade_no"],
		Amount:   amount,
		TradeNo:  params["trade_no"],
	}, nil
}

// verifyAliSign 用支付宝公钥对回调参数做 RSA2（PKCS1v15 + SHA256）验签。
// 签名串：按 key 字母序拼接 key=value，剔除 sign 与空值，与下单签名 buildAliSignStr 同约定。
func verifyAliSign(params map[string]string, publicKeyPEM, signatureBase64 string) bool {
	// 1. 解析 PEM
	fixed := fixPEM(publicKeyPEM, "PUBLIC KEY")
	block, _ := pem.Decode([]byte(fixed))
	if block == nil {
		slog.Error("[支付宝] PEM 解析失败", "pemLen", len(publicKeyPEM), "fixedLen", len(fixed))
		return false
	}

	// 2. 解析公钥（先尝试 PKIX，再尝试 PKCS1）
	var pub *rsa.PublicKey
	pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		// 支付宝公钥可能是 PKCS#1 格式，尝试解析
		pubPKCS1, err2 := x509.ParsePKCS1PublicKey(block.Bytes)
		if err2 != nil {
			slog.Error("[支付宝] 公钥解析失败", "pkixErr", err.Error(), "pkcs1Err", err2.Error())
			return false
		}
		pub = pubPKCS1
	} else {
		var ok bool
		pub, ok = pubAny.(*rsa.PublicKey)
		if !ok {
			slog.Error("[支付宝] 公钥类型不是 RSA", "type", fmt.Sprintf("%T", pubAny))
			return false
		}
	}

	// 3. 解码签名
	sig, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		slog.Error("[支付宝] 签名 base64 解码失败", "signLen", len(signatureBase64), "err", err)
		return false
	}

	// 4. 验签（先按官方 SDK 规则剔除 sign_type，失败则回退尝试包含 sign_type）
	h := sha256.Sum256([]byte(buildAliVerifyStr(params)))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, h[:], sig); err != nil {
		// 回退：尝试包含 sign_type 的签名串（兼容不同支付宝版本）
		h2 := sha256.Sum256([]byte(buildAliSignStr(params)))
		if err2 := rsa.VerifyPKCS1v15(pub, crypto.SHA256, h2[:], sig); err2 != nil {
			slog.Error("[支付宝] RSA 验签不通过", "signStrNoType", buildAliVerifyStr(params)[:80], "signStrWithType", buildAliSignStr(params)[:80])
			return false
		}
	}
	return true
}

func buildAliSignStr(p map[string]string) string {
	// signature string: sort keys alphabetically, join key=value, exclude sign and empty values.
	// sign_type inclusion depends on context: included for payment creation, excluded for callback verification.
	// This function is for payment creation - keep sign_type.
	keys := make([]string, 0, len(p))
	for k := range p {
		if k == "sign" || p[k] == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, k+"="+p[k])
	}
	return strings.Join(parts, "&")
}

// buildAliVerifyStr builds verification sign string for callbacks: same as buildAliSignStr but also excludes sign_type
func buildAliVerifyStr(p map[string]string) string {
	keys := make([]string, 0, len(p))
	for k := range p {
		if k == "sign" || k == "sign_type" || p[k] == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, k+"="+p[k])
	}
	return strings.Join(parts, "&")
}

func encodeParams(p map[string]string) string {
	v := url.Values{}
	for k, val := range p { v.Set(k, val) }
	return v.Encode()
}

func signRSABase64(data, privateKeyPEM string) (string, error) {
	block, _ := pem.Decode([]byte(fixPEM(privateKeyPEM, "RSA PRIVATE KEY")))
	if block == nil { return "", fmt.Errorf("无法解析私钥") }
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil { return "", err }
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok { return "", fmt.Errorf("不是 RSA 私钥") }
	h := sha256.Sum256([]byte(data))
	sig, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, h[:])
	if err != nil { return "", err }
	return base64.StdEncoding.EncodeToString(sig), nil
}
