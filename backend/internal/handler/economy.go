package handler

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"xzji/backend/internal/middleware"
	"xzji/backend/internal/models"
	"xzji/backend/internal/service"
	"xzji/backend/internal/service/payment"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// economySvc 从 context 获取 EconomyService（依赖注入）
func economySvc(c *gin.Context) *service.EconomyService {
	return getApp(c).EconomySvc
}

// processPaidOrderWithLevelUp 包装 ProcessPaidOrder，事务提交后触发等级升级检查
// （对齐原项目：事务外执行，异常不影响支付结果）
func processPaidOrderWithLevelUp(db *gorm.DB, orderNo, paidAmount, tradeNo string) (map[string]any, error) {
	userID, result, err := payment.ProcessPaidOrder(db, orderNo, paidAmount, tradeNo)
	if err != nil {
		return result, err
	}
	if userID > 0 {
		service.CheckLevelUp(db, userID)
	}
	return result, nil
}

func PaymentMethods(c *gin.Context) {
	app := getApp(c)
	cfg, _ := app.Cache.GetPaymentConfig(app.DB)
	labels := map[string]string{"wechat": "微信支付", "alipay": "支付宝", "epay": "易支付"}
	out := make([]map[string]any, 0, len(cfg.EnabledProviders))
	for _, name := range cfg.EnabledProviders {
		// 过滤掉未注册的支付方式（如已移除的 mock 等）
		if _, err := payment.Get(name); err != nil {
			continue
		}
		label := labels[name]
		if label == "" {
			label = name
		}
		out = append(out, map[string]any{"name": name, "label": label})
	}
	c.JSON(http.StatusOK, out)
}

func Recharge(c *gin.Context) {
	var req service.RechargeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}
	res, err := economySvc(c).Recharge(middleware.GetUserID(c), req, c.GetHeader("User-Agent"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func RechargeStatus(c *gin.Context) {
	uid := middleware.GetUserID(c)
	var order models.PaymentOrder
	if err := getApp(c).DB.Where("order_no = ? AND user_id = ?", c.Param("orderNo"), uid).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "订单不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"orderNo": order.OrderNo, "amount": order.Amount, "status": order.Status, "points": order.Points, "paymentMethod": order.PaymentMethod, "paidAt": order.PaidAt})
}

func PurchaseLink(c *gin.Context) {
	var req service.PurchaseLinkReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不完整"})
		return
	}
	res, err := economySvc(c).PurchaseLink(middleware.GetUserID(c), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func PurchaseStatus(c *gin.Context) {
	kid, _ := strconv.Atoi(c.Query("keywordId"))
	bIdx, _ := strconv.Atoi(c.Query("blockIndex"))
	var cnt int64
	getApp(c).DB.Model(&models.LinkPurchase{}).Where("user_id = ? AND keyword_id = ? AND block_index = ?", middleware.GetUserID(c), kid, bIdx).Count(&cnt)
	resp := gin.H{"purchased": cnt > 0}
	// 已购买：下发真实 url/code（鉴权后），供前端直接打开，无需再调 purchase-link
	if cnt > 0 {
		url, code, err := service.ResolveBlock(getApp(c).DB, uint(kid), bIdx)
		if err == nil {
			resp["url"] = url
			resp["code"] = code
		}
	}
	c.JSON(http.StatusOK, resp)
}

func GetMembership(c *gin.Context) {
	res, err := economySvc(c).GetMembership(middleware.GetUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func BuyMembership(c *gin.Context) {
	var req service.BuyMembershipReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择会员等级"})
		return
	}
	res, err := economySvc(c).BuyMembership(middleware.GetUserID(c), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func GetTransactions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	res, err := economySvc(c).Transactions(middleware.GetUserID(c), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// === Admin 支付订单 ===

// adminOrderRow 订单列表行（camelCase 字段对齐原版前端契约，含用户名/头像）
type adminOrderRow struct {
	ID            uint       `json:"id"`
	OrderNo       string     `json:"orderNo"`
	UserID        uint       `json:"userId"`
	Amount        float64    `json:"amount"`
	Points        int        `json:"points"`
	Status        string     `json:"status"`
	PaymentMethod string     `json:"paymentMethod"`
	PaymentScene  string     `json:"paymentScene"`
	TradeNo       string     `json:"tradeNo"`
	CreatedAt     time.Time  `json:"createdAt"`
	PaidAt        *time.Time `json:"paidAt"`
	Username      string     `json:"username"`
	Avatar        string     `json:"avatar"`
}

func adminOrderFilters(q *gorm.DB, status, method, search string) *gorm.DB {
	if status != "all" && status != "" {
		q = q.Where("payment_orders.status = ?", status)
	}
	if method != "all" && method != "" {
		q = q.Where("payment_orders.payment_method = ?", method)
	}
	if search != "" {
		q = q.Where("payment_orders.order_no LIKE ? OR users.username LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	return q
}

func AdminPaymentOrdersList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	status := c.DefaultQuery("status", "all")
	method := c.DefaultQuery("method", "all")
	search := c.Query("search")

	join := "LEFT JOIN users ON users.id = payment_orders.user_id"

	var total int64
	countQ := getApp(c).DB.Table("payment_orders").Joins(join)
	countQ = adminOrderFilters(countQ, status, method, search)
	countQ.Count(&total)

	var rows []adminOrderRow
	listQ := getApp(c).DB.Table("payment_orders").
		Select("payment_orders.id, payment_orders.order_no, payment_orders.user_id, payment_orders.amount, payment_orders.points, payment_orders.status, payment_orders.payment_method, payment_orders.payment_scene, payment_orders.trade_no, payment_orders.created_at, payment_orders.paid_at, users.username, users.avatar").
		Joins(join)
	listQ = adminOrderFilters(listQ, status, method, search)
	listQ.Order("payment_orders.id DESC").Offset((page - 1) * limit).Limit(limit).Scan(&rows)
	if rows == nil {
		// 避免 Go nil 切片序列化为 null（前端访问 .length 报错），返回空数组
		rows = []adminOrderRow{}
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}
	c.JSON(http.StatusOK, gin.H{
		"list": rows,
		"pagination": gin.H{"page": page, "limit": limit, "total": total, "totalPages": totalPages},
	})
}

func AdminPaymentOrdersStats(c *gin.Context) {
	var total, paid, pending int64
	var revenue float64
	db := getApp(c).DB
	db.Model(&models.PaymentOrder{}).Count(&total)
	db.Model(&models.PaymentOrder{}).Where("status = 'paid'").Count(&paid)
	db.Model(&models.PaymentOrder{}).Where("status = 'pending'").Count(&pending)
	db.Model(&models.PaymentOrder{}).Where("status = 'paid'").Select("COALESCE(SUM(amount), 0)").Scan(&revenue)
	c.JSON(http.StatusOK, gin.H{
		"total": total, "paidCount": paid, "pendingCount": pending, "totalRevenue": revenue,
	})
}

func AdminPaymentOrderBatchDelete(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if len(req.IDs) > 0 {
		getApp(c).DB.Where("id IN ?", req.IDs).Delete(&models.PaymentOrder{})
	}
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("已删除 %d 条订单", len(req.IDs))})
}

func AdminPaymentOrderDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	getApp(c).DB.Delete(&models.PaymentOrder{}, id)
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// === 支付回调 ===

func PaymentCallbackWechat(c *gin.Context) {
	provider, err := payment.Get("wechat")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": "provider not found"})
		return
	}
	app := getApp(c)
	cfg, _ := app.Cache.GetPaymentConfig(app.DB)
	body, _ := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20)) // 限制 1MB
	result, _ := provider.VerifyCallback(payment.CallbackRequest{Method: "POST", Body: body}, cfg)
	if result.Skip {
		c.JSON(http.StatusOK, gin.H{"code": "SUCCESS"})
		return
	}
	if !result.Verified {
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": result.Error})
		return
	}
			if result.OrderNo != "" {
				if _, err := processPaidOrderWithLevelUp(getApp(c).DB, result.OrderNo, fmt.Sprintf("%.2f", result.Amount), result.TradeNo); err != nil {
					slog.Error("[微信] 入账失败", "orderNo", result.OrderNo, "err", err)
					var existing models.PaymentOrder
					if getApp(c).DB.Where("order_no = ? AND status = ?", result.OrderNo, "paid").First(&existing).Error == nil {
						c.JSON(http.StatusOK, gin.H{"code": "SUCCESS"})
						return
					}
					c.JSON(http.StatusOK, gin.H{"code": "SUCCESS"})
					return
				}
			}
	c.JSON(http.StatusOK, gin.H{"code": "SUCCESS"})
}

func PaymentCallbackAlipay(c *gin.Context) {
	// 支付宝异步通知为 POST 表单（application/x-www-form-urlencoded）
	// 注意：不能直接用 c.Request.ParseForm()，因为 form parser 会把 base64
	// 签名中的 + 转成空格，导致验签失败。这里手动解析 body，对 sign 参数
	// 使用 url.PathUnescape（不转换 +），其余参数正常解码。
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20)) // 限制 1MB
	if err != nil {
		slog.Error("[支付宝] 读取回调 body 失败", "err", err)
		c.String(http.StatusOK, "fail")
		return
	}
	params := map[string]string{}
	for _, pair := range strings.Split(string(body), "&") {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key, _ := url.QueryUnescape(kv[0])
		var val string
		if key == "sign" {
			// sign 必须用 PathUnescape：保留 base64 中的 + 符号
			val, _ = url.PathUnescape(kv[1])
		} else {
			val, _ = url.QueryUnescape(kv[1])
		}
		params[key] = val
	}

		provider, err := payment.Get("alipay")
		if err != nil {
			slog.Error("[支付宝] 支付 provider 未注册", "err", err)
			c.String(http.StatusOK, "fail")
			return
		}
		app := getApp(c)
		cfg, _ := app.Cache.GetPaymentConfig(app.DB)
		result, _ := provider.VerifyCallback(payment.CallbackRequest{Method: c.Request.Method, Query: params}, cfg)
		if result.Skip {
			c.String(http.StatusOK, "success")
			return
		}
		if !result.Verified {
			slog.Error("[支付宝] 回调验签失败", "orderNo", params["out_trade_no"], "error", result.Error)
			c.String(http.StatusOK, "fail")
			return
		}
		if result.OrderNo != "" {
			if _, err := processPaidOrderWithLevelUp(getApp(c).DB, result.OrderNo, fmt.Sprintf("%.2f", result.Amount), result.TradeNo); err != nil {
				var existing models.PaymentOrder
				if getApp(c).DB.Where("order_no = ? AND status = ?", result.OrderNo, "paid").First(&existing).Error == nil {
					// 重复通知，订单已支付，返回 success 阻止重试
				} else {
					slog.Error("[支付宝] 入账失败，订单不存在", "orderNo", result.OrderNo, "err", err)
				}
				c.String(http.StatusOK, "success")
				return
			}
		}
	c.String(http.StatusOK, "success")
}

func PaymentCallbackEpay(c *gin.Context) {
	provider, err := payment.Get("epay")
	if err != nil { c.String(http.StatusOK, "fail"); return }
	app := getApp(c)
	cfg, _ := app.Cache.GetPaymentConfig(app.DB)
	query := map[string]string{}
	for k, v := range c.Request.URL.Query() { query[k] = v[0] }
	result, _ := provider.VerifyCallback(payment.CallbackRequest{Method: c.Request.Method, Query: query}, cfg)
	if result.Skip { c.String(http.StatusOK, "success"); return }
	if !result.Verified { c.String(http.StatusOK, "fail"); return }
	if result.OrderNo != "" {
				if _, err := processPaidOrderWithLevelUp(getApp(c).DB, result.OrderNo, fmt.Sprintf("%.2f", result.Amount), result.TradeNo); err != nil {
					slog.Error("[易支付] 入账失败", "orderNo", result.OrderNo, "err", err)
					c.String(http.StatusOK, "success")
					return
				}
			}
	c.String(http.StatusOK, "success")
}
