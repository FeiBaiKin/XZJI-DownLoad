package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"
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
	if payMethod == "alipay" { scene = cfg.AliScene; if scene == "" { scene = "page" } }
	if payMethod == "epay" { scene = "v1" }

	order := models.PaymentOrder{
		OrderNo: orderNo, UserID: userID, Amount: req.Amount, Points: bonusExp,
		Status: "pending", PaymentMethod: payMethod, PaymentScene: scene,
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
	Access      string  `json:"access"` // "balance" or "membership"
}

// resolveLinkURL 取付费链接的真实地址与提取码（仅在购买/会员校验通过后调用）。
// 解析失败则返回空串（不阻断已成功的购买流程）。
func resolveLinkURL(db *gorm.DB, keywordID uint, blockIndex int) (url, code string) {
	url, code, _ = ResolveBlock(db, keywordID, blockIndex)
	return
}

func (s *EconomyService) PurchaseLink(userID uint, req PurchaseLinkReq) (map[string]any, error) {
	// 检查是否已购买
	var cnt int64
	s.DB.Model(&models.LinkPurchase{}).Where("user_id = ? AND keyword_id = ? AND block_index = ?", userID, req.KeywordID, req.BlockIndex).Count(&cnt)
	if cnt > 0 {
		today := time.Now().UTC().Format("2006-01-02")
		s.DB.Exec("UPDATE users SET daily_downloads = IF(download_date = ?, daily_downloads + 1, 1), download_date = ? WHERE id = ?", today, today, userID)
		url, code := resolveLinkURL(s.DB, req.KeywordID, req.BlockIndex)
		return map[string]any{"message": "已购买", "alreadyPurchased": true, "url": url, "code": code}, nil
	}
	// 检查会员
	var user models.User
	s.DB.First(&user, userID)
	membership := checkMembershipInternal(&user)
	isMembershipLink := req.Access == "membership"

	if isMembershipLink && membership != nil {
		today := time.Now().UTC().Format("2006-01-02")
		var cfg models.MembershipConfig
		if err := s.DB.Where("level_name = ?", membership.Level).First(&cfg).Error; err == nil {
			if cfg.DailyDownloadLimit >= 0 && user.DailyDownloads >= cfg.DailyDownloadLimit {
				return nil, fmt.Errorf("今日下载次数已达上限（%d次）", cfg.DailyDownloadLimit)
			}
		}
		s.DB.Create(&models.LinkPurchase{UserID: userID, KeywordID: req.KeywordID, BlockIndex: req.BlockIndex, BalanceSpent: 0})
		s.DB.Exec("UPDATE users SET daily_downloads = IF(download_date = ?, daily_downloads + 1, 1), download_date = ? WHERE id = ?", today, today, userID)
		url, code := resolveLinkURL(s.DB, req.KeywordID, req.BlockIndex)
		return map[string]any{"message": "会员免费访问", "membership": true, "url": url, "code": code}, nil
	}
	if isMembershipLink && membership == nil {
		return nil, fmt.Errorf("此链接仅限会员访问")
	}
	// 余额购买（事务保护）
	cost := req.BalanceCost
	if cost <= 0 { return nil, fmt.Errorf("该链接需要付费购买") }
	finalCost := cost
	var discount float64
	if membership != nil {
		var cfg models.MembershipConfig
		if err := s.DB.Where("level_name = ?", membership.Level).First(&cfg).Error; err == nil && cfg.Discount > 0 && cfg.Discount < 1 {
			finalCost = math.Round(cost*cfg.Discount*100) / 100
			discount = cfg.Discount
		}
	}
	var updated models.User
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Exec("UPDATE users SET balance = balance - ? WHERE id = ? AND balance >= ?", finalCost, userID, finalCost)
		if res.RowsAffected == 0 { return fmt.Errorf("余额不足") }
		tx.Create(&models.LinkPurchase{UserID: userID, KeywordID: req.KeywordID, BlockIndex: req.BlockIndex, BalanceSpent: finalCost})
		desc := fmt.Sprintf("购买链接访问权限")
		if discount > 0 { desc += fmt.Sprintf(" (会员%d折)", int(discount*100)) }
		tx.Create(&models.PointsTransaction{UserID: userID, Type: "purchase", Amount: -finalCost, Description: desc, RefID: &req.KeywordID})
		today := time.Now().UTC().Format("2006-01-02")
		tx.Exec("UPDATE users SET daily_downloads = IF(download_date = ?, daily_downloads + 1, 1), download_date = ? WHERE id = ?", today, today, userID)
		return tx.First(&updated, userID).Error
	})
	if err != nil { return nil, err }
	url, code := resolveLinkURL(s.DB, req.KeywordID, req.BlockIndex)
	return map[string]any{
		"message": "购买成功", "balance": updated.Balance, "balanceSpent": finalCost,
		"originalCost": cost, "discountApplied": discount,
		"url": url, "code": code,
	}, nil
}

// ====== 会员 ======

type BuyMembershipReq struct {
	Level string `json:"level"`
}

func (s *EconomyService) BuyMembership(userID uint, req BuyMembershipReq) (map[string]any, error) {
	var cfg models.MembershipConfig
	if err := s.DB.Where("level_name = ?", req.Level).First(&cfg).Error; err != nil {
		return nil, fmt.Errorf("无效的会员等级")
	}
	var user models.User
	s.DB.First(&user, userID)
	if user.Balance < cfg.Price {
		return nil, fmt.Errorf("余额不足，请先充值")
	}
	base := time.Now()
	if user.MembershipExpiresAt != nil && user.MembershipExpiresAt.After(base) {
		base = *user.MembershipExpiresAt
	}
	expiresAt := base.AddDate(0, 0, cfg.Days)
	orderNo := fmt.Sprintf("M%d%s", time.Now().UnixMilli(), randStr(4))

	var updated models.User
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Exec("UPDATE users SET balance = balance - ?, membership_level = ?, membership_expires_at = ? WHERE id = ? AND balance >= ?",
			cfg.Price, req.Level, expiresAt, userID, cfg.Price)
		if res.RowsAffected == 0 { return fmt.Errorf("余额不足") }
		tx.Create(&models.MembershipOrder{OrderNo: orderNo, UserID: userID, MembershipLevel: req.Level, Amount: cfg.Price, Days: cfg.Days, Status: "paid", PaidAt: nowPtr()})
		tx.Create(&models.PointsTransaction{UserID: userID, Type: "membership", Amount: -cfg.Price, Description: "购买" + req.Level})
		return tx.First(&updated, userID).Error
	})
	if err != nil { return nil, err }
	return map[string]any{
		"message": "购买成功", "balance": updated.Balance,
		"membershipLevel": updated.MembershipLevel, "membershipExpiresAt": updated.MembershipExpiresAt,
	}, nil
}

func (s *EconomyService) GetMembership(userID uint) (map[string]any, error) {
	var user models.User
	s.DB.First(&user, userID)
	m := checkMembershipInternal(&user)
	today := time.Now().UTC().Format("2006-01-02")
	if user.DownloadDate == nil || user.DownloadDate.UTC().Format("2006-01-02") != today {
		s.DB.Model(&user).Updates(map[string]any{"daily_downloads": 0, "download_date": today})
	}
	var configs []models.MembershipConfig
	s.DB.Order("price ASC").Find(&configs)
	dailyLimit := 0
	if m != nil {
		for _, c := range configs { if c.LevelName == m.Level { dailyLimit = c.DailyDownloadLimit; break } }
	}
	return map[string]any{"membership": m, "downloads": user.DailyDownloads, "dailyLimit": dailyLimit, "configs": configs}, nil
}

type memberBrief struct {
	Level     string     `json:"level"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

func checkMembershipInternal(u *models.User) *memberBrief {
	if u.MembershipLevel == nil || u.MembershipExpiresAt == nil { return nil }
	if time.Now().After(*u.MembershipExpiresAt) { return nil }
	return &memberBrief{Level: *u.MembershipLevel, ExpiresAt: u.MembershipExpiresAt}
}

// ====== 交易记录 ======

func (s *EconomyService) Transactions(userID uint, page, limit int) (map[string]any, error) {
	if page < 1 { page = 1 }
	if limit < 1 || limit > 100 { limit = 20 }
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
