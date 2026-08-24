package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"xzji/backend/internal/models"
	"xzji/backend/internal/service/payment"
)

type EconomyService struct{ DB *gorm.DB }

// ====== 充值 ======

type RechargeReq struct {
	Amount float64 `json:"amount"`
	Method string  `json:"method"` // wechat / alipay / epay
}

func (s *EconomyService) Recharge(userID uint, req RechargeReq, userAgent string) (any, error) {
	if req.Amount <= 0 || req.Amount > 50000 {
		return nil, fmt.Errorf("充值金额无效")
	}
	bonusExp := int(math.Floor(req.Amount * 2))
	orderNo := fmt.Sprintf("R%d%s", time.Now().UnixMilli(), randStr(6))
	payMethod := req.Method
	if payMethod == "" {
		return nil, fmt.Errorf("请选择支付方式")
	}

	cfg := payment.GetPaymentConfig(s.DB)
	// 校验支付方式已在后台开启
	enabled := false
	for _, e := range cfg.EnabledProviders {
		if e == payMethod {
			enabled = true
			break
		}
	}
	if !enabled {
		return nil, fmt.Errorf("该支付方式未开启")
	}
	provider, err := payment.Get(payMethod)
	if err != nil {
		return nil, err
	}
	scene := "native"
	if payMethod == "alipay" {
		scene = cfg.AliScene
		if scene == "" {
			scene = "page"
		}
	}
	if payMethod == "epay" {
		scene = "v1"
	}

	order := models.PaymentOrder{
		OrderNo: orderNo, UserID: userID, Amount: req.Amount, Points: bonusExp,
		Status: "pending", PaymentMethod: payMethod, PaymentScene: scene,
		OrderType: models.OrderTypeRecharge,
	}
	if err := s.DB.Create(&order).Error; err != nil {
		return nil, fmt.Errorf("创建订单失败")
	}

	extra := map[string]string{}
	if userAgent != "" {
		extra["User-Agent"] = userAgent
	}
	result, err := provider.CreatePayment(payment.Order{OrderNo: orderNo, Amount: req.Amount, Points: bonusExp, UserID: userID}, cfg, scene, extra)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"type":          result.Type,
		"scene":         result.Scene,
		"payUrl":        result.PayUrl,
		"qrCode":        result.QRCode,
		"orderNo":       orderNo,
		"amount":        req.Amount,
		"bonusExp":      bonusExp,
		"paymentMethod": payMethod,
	}, nil
}

// ====== 链接购买 ======

type PurchaseLinkReq struct {
	KeywordID   uint    `json:"keywordId"`
	BlockIndex  int     `json:"blockIndex"`
	BalanceCost float64 `json:"balanceCost"`
	Access      string  `json:"access"`
}

// resolveLinkURL 取付费链接的真实地址与提取码（仅在购买/会员校验通过后调用）。
// 解析失败则返回空串（不阻断已成功的购买流程）。
func resolveLinkURL(db *gorm.DB, keywordID uint, blockIndex int) (url, code string) {
	url, code, _ = ResolveBlock(db, keywordID, blockIndex)
	return
}

func (s *EconomyService) PurchaseLink(userID uint, req PurchaseLinkReq) (map[string]any, error) {
	block, err := ResolveLinkBlock(s.DB, req.KeywordID, req.BlockIndex)
	if err != nil {
		return nil, err
	}
	if block.Access == "free" {
		return map[string]any{"message": "免费链接", "url": block.URL, "code": block.Code}, nil
	}

	if block.Access == "membership" {
		err := s.DB.Transaction(func(tx *gorm.DB) error {
			var user models.User
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
				return fmt.Errorf("用户不存在")
			}
			membership := checkMembershipInternal(&user)
			if membership == nil {
				return fmt.Errorf("此链接仅限会员访问")
			}
			var cfg models.MembershipConfig
			if err := tx.Where("level_name = ?", membership.Level).First(&cfg).Error; err == nil {
				downloadsToday := downloadsTodayOf(&user)
				if cfg.DailyDownloadLimit >= 0 && downloadsToday >= cfg.DailyDownloadLimit {
					return fmt.Errorf("今日下载次数已达上限（%d次）", cfg.DailyDownloadLimit)
				}
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.LinkPurchase{UserID: userID, KeywordID: req.KeywordID, BlockIndex: req.BlockIndex, BalanceSpent: 0}).Error; err != nil {
				return fmt.Errorf("记录访问权限失败")
			}
			return incrementDailyDownloads(tx, userID)
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"message": "会员免费访问", "membership": true, "url": block.URL, "code": block.Code}, nil
	}

	if block.Access != "balance" {
		return nil, fmt.Errorf("链接权限配置无效")
	}
	if block.Cost <= 0 {
		return nil, fmt.Errorf("该链接需要付费购买")
	}

	cost := block.Cost
	var discount float64
	finalCost := cost
	var updated models.User
	alreadyPurchased := false
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			return fmt.Errorf("用户不存在")
		}
		var cnt int64
		if err := tx.Model(&models.LinkPurchase{}).Where("user_id = ? AND keyword_id = ? AND block_index = ?", userID, req.KeywordID, req.BlockIndex).Count(&cnt).Error; err != nil {
			return err
		}
		if cnt > 0 {
			alreadyPurchased = true
			if err := incrementDailyDownloads(tx, userID); err != nil {
				return err
			}
			return tx.First(&updated, userID).Error
		}
		membership := checkMembershipInternal(&user)
		if membership != nil {
			var cfg models.MembershipConfig
			if err := tx.Where("level_name = ?", membership.Level).First(&cfg).Error; err == nil && cfg.Discount > 0 && cfg.Discount < 1 {
				finalCost = math.Round(cost*cfg.Discount*100) / 100
				discount = cfg.Discount
			}
		}
		res := tx.Exec("UPDATE users SET balance = balance - ? WHERE id = ? AND balance >= ?", finalCost, userID, finalCost)
		if res.RowsAffected == 0 {
			return fmt.Errorf("余额不足")
		}
		if err := tx.Create(&models.LinkPurchase{UserID: userID, KeywordID: req.KeywordID, BlockIndex: req.BlockIndex, BalanceSpent: finalCost}).Error; err != nil {
			return fmt.Errorf("记录购买失败")
		}
		desc := fmt.Sprintf("购买链接访问权限")
		if discount > 0 {
			desc += fmt.Sprintf(" (会员%d折)", int(discount*100))
		}
		if err := tx.Create(&models.PointsTransaction{UserID: userID, Type: "purchase", Amount: -finalCost, Description: desc, RefID: &req.KeywordID}).Error; err != nil {
			return fmt.Errorf("创建交易流水失败")
		}
		if err := incrementDailyDownloads(tx, userID); err != nil {
			return err
		}
		return tx.First(&updated, userID).Error
	})
	if err != nil {
		return nil, err
	}
	message := "购买成功"
	if alreadyPurchased {
		message = "已购买"
	}
	return map[string]any{
		"message": message, "alreadyPurchased": alreadyPurchased, "balance": updated.Balance, "balanceSpent": finalCost,
		"originalCost": cost, "discountApplied": discount,
		"url": block.URL, "code": block.Code,
	}, nil
}

// ====== 会员 ======

type BuyMembershipReq struct {
	Level string `json:"level"`
	// Method 空 / "balance" 走余额扣费；wechat / alipay / epay 走第三方支付直付
	Method string `json:"method"`
}

func (s *EconomyService) BuyMembership(userID uint, req BuyMembershipReq, userAgent string) (map[string]any, error) {
	var cfg models.MembershipConfig
	if err := s.DB.Where("level_name = ?", req.Level).First(&cfg).Error; err != nil {
		return nil, fmt.Errorf("无效的会员等级")
	}
	if cfg.Price < 0 {
		return nil, fmt.Errorf("无效的会员价格")
	}
	if req.Method != "" && req.Method != "balance" {
		return s.buyMembershipByPayment(userID, cfg, req.Method, userAgent)
	}
	orderNo := fmt.Sprintf("M%d%s", time.Now().UnixMilli(), randStr(4))

	var updated models.User
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		// 锁用户行后重算到期时间，避免并发购买双扣费只延一次
		var user models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			return fmt.Errorf("用户不存在")
		}
		if user.Balance < cfg.Price {
			return fmt.Errorf("余额不足，请先充值")
		}
		base := time.Now()
		if user.MembershipExpiresAt != nil && user.MembershipExpiresAt.After(base) {
			base = *user.MembershipExpiresAt
		}
		expiresAt := base.AddDate(0, 0, cfg.Days)

		res := tx.Exec("UPDATE users SET balance = balance - ?, membership_level = ?, membership_expires_at = ? WHERE id = ? AND balance >= ?",
			cfg.Price, req.Level, expiresAt, userID, cfg.Price)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("余额不足")
		}
		if err := tx.Create(&models.MembershipOrder{
			OrderNo: orderNo, UserID: userID, MembershipLevel: req.Level,
			Amount: cfg.Price, Days: cfg.Days, Status: "paid", PaidAt: nowPtr(),
		}).Error; err != nil {
			return fmt.Errorf("创建会员订单失败")
		}
		if err := tx.Create(&models.PointsTransaction{
			UserID: userID, Type: "membership", Amount: -cfg.Price, Description: "购买" + req.Level,
		}).Error; err != nil {
			return fmt.Errorf("创建流水失败")
		}
		return tx.First(&updated, userID).Error
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"message": "购买成功", "balance": updated.Balance,
		"membershipLevel": updated.MembershipLevel, "membershipExpiresAt": updated.MembershipExpiresAt,
	}, nil
}

// buyMembershipByPayment 会员直付：下一笔 order_type=membership 的支付订单，
// 付款回调（payment.ProcessPaidOrder）里再开通会员，用户无需先充值余额。
// 返回结构与 Recharge 对齐，前端可直接复用扫码 / 跳转 / 轮询那套流程。
func (s *EconomyService) buyMembershipByPayment(userID uint, cfg models.MembershipConfig, method, userAgent string) (map[string]any, error) {
	if cfg.Price <= 0 {
		return nil, fmt.Errorf("该档位无需付费，请使用余额开通")
	}
	payCfg := payment.GetPaymentConfig(s.DB)
	enabled := false
	for _, e := range payCfg.EnabledProviders {
		if e == method {
			enabled = true
			break
		}
	}
	if !enabled {
		return nil, fmt.Errorf("该支付方式未开启")
	}
	provider, err := payment.Get(method)
	if err != nil {
		return nil, err
	}
	scene := "native"
	if method == "alipay" {
		scene = payCfg.AliScene
		if scene == "" {
			scene = "page"
		}
	}
	if method == "epay" {
		scene = "v1"
	}

	// 与充值同口径赠经验，保证「直付」与「充值同额再买」拿到的收益一致
	bonusExp := int(math.Floor(cfg.Price * 2))
	orderNo := fmt.Sprintf("MP%d%s", time.Now().UnixMilli(), randStr(6))
	order := models.PaymentOrder{
		OrderNo: orderNo, UserID: userID, Amount: cfg.Price, Points: bonusExp,
		Status: "pending", PaymentMethod: method, PaymentScene: scene,
		OrderType: models.OrderTypeMembership, MembershipLevel: cfg.LevelName, MembershipDays: cfg.Days,
	}
	if err := s.DB.Create(&order).Error; err != nil {
		return nil, fmt.Errorf("创建订单失败")
	}

	extra := map[string]string{}
	if userAgent != "" {
		extra["User-Agent"] = userAgent
	}
	result, err := provider.CreatePayment(payment.Order{
		OrderNo: orderNo, Amount: cfg.Price, Points: bonusExp, UserID: userID,
		Subject: fmt.Sprintf("开通%s %.2f元", cfg.LevelName, cfg.Price),
	}, payCfg, scene, extra)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"pay":             true,
		"type":            result.Type,
		"scene":           result.Scene,
		"payUrl":          result.PayUrl,
		"qrCode":          result.QRCode,
		"orderNo":         orderNo,
		"amount":          cfg.Price,
		"bonusExp":        bonusExp,
		"paymentMethod":   method,
		"membershipLevel": cfg.LevelName,
	}, nil
}

func (s *EconomyService) GetMembership(userID uint) (map[string]any, error) {
	var user models.User
	s.DB.First(&user, userID)
	m := checkMembershipInternal(&user)
	// 日界线口径同 downloadsTodayOf：本地时区 + DATE 列直接 Format
	today := time.Now().Format("2006-01-02")
	downloads := downloadsTodayOf(&user)
	if user.DownloadDate == nil || user.DownloadDate.Format("2006-01-02") != today {
		s.DB.Model(&user).Updates(map[string]any{"daily_downloads": 0, "download_date": today})
	}
	var configs []models.MembershipConfig
	s.DB.Order("price ASC").Find(&configs)
	dailyLimit := 0
	if m != nil {
		for _, c := range configs {
			if c.LevelName == m.Level {
				dailyLimit = c.DailyDownloadLimit
				break
			}
		}
	}
	return map[string]any{"membership": m, "downloads": downloads, "dailyLimit": dailyLimit, "configs": configs}, nil
}

type memberBrief struct {
	Level     string     `json:"level"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

func checkMembershipInternal(u *models.User) *memberBrief {
	if u.MembershipLevel == nil || u.MembershipExpiresAt == nil {
		return nil
	}
	if time.Now().After(*u.MembershipExpiresAt) {
		return nil
	}
	return &memberBrief{Level: *u.MembershipLevel, ExpiresAt: u.MembershipExpiresAt}
}

func downloadsTodayOf(u *models.User) int {
	// 日界线用服务器本地时区，与 incrementDailyDownloads 写入口径一致；
	// DATE 列被驱动按 DSN loc 解析为本地零点，直接 Format 还原存储日期，禁用 .UTC()（东八区会回退一天）
	today := time.Now().Format("2006-01-02")
	if u.DownloadDate != nil && u.DownloadDate.Format("2006-01-02") == today {
		return u.DailyDownloads
	}
	return 0
}

func incrementDailyDownloads(db *gorm.DB, userID uint) error {
	today := time.Now().Format("2006-01-02")
	return db.Exec("UPDATE users SET daily_downloads = CASE WHEN download_date = ? THEN daily_downloads + 1 ELSE 1 END, download_date = ? WHERE id = ?", today, today, userID).Error
}

// ====== 交易记录 ======

func (s *EconomyService) Transactions(userID uint, page, limit int) (map[string]any, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	var total int64
	s.DB.Model(&models.PointsTransaction{}).Where("user_id = ?", userID).Count(&total)
	var list []models.PointsTransaction
	s.DB.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Offset((page - 1) * limit).Find(&list)
	return map[string]any{"transactions": list, "total": total, "page": page, "limit": limit}, nil
}

// ====== 辅助 ======

func randStr(n int) string {
	b := make([]byte, n/2+1)
	rand.Read(b)
	return hex.EncodeToString(b)[:n]
}

func nowPtr() *time.Time { t := time.Now(); return &t }
