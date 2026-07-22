package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// rateLimiter 基于内存的滑动窗口限流器。
// 按客户端 IP 统计请求次数，超出阈值时返回 429 Too Many Requests。
type rateLimiter struct {
	mu       sync.Mutex
	records  map[string][]time.Time // IP → 请求时间戳列表
	maxReqs  int                    // 窗口内最大请求数
	window   time.Duration          // 滑动窗口大小
}

func newRateLimiter(maxReqs int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		records: make(map[string][]time.Time),
		maxReqs: maxReqs,
		window:  window,
	}
	// 后台清理过期记录，防止内存泄漏
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, times := range rl.records {
			valid := times[:0]
			for _, t := range times {
				if now.Sub(t) <= rl.window {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(rl.records, ip)
			} else {
				rl.records[ip] = valid
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	times := rl.records[ip]

	// 过滤掉窗口外的记录
	valid := times[:0]
	for _, t := range times {
		if now.Sub(t) <= rl.window {
			valid = append(valid, t)
		}
	}
	rl.records[ip] = valid

	if len(valid) >= rl.maxReqs {
		return false
	}

	rl.records[ip] = append(rl.records[ip], now)
	return true
}

// clientIP 从请求中提取客户端 IP。
// 默认仅使用 RemoteAddr，避免客户端伪造 X-Forwarded-For / X-Real-IP 绕过限流。
// 若部署在受信反向代理后，请在启动时调用 gin.Engine.SetTrustedProxies，
// 并改用 c.ClientIP()（其会按受信代理列表解析转发头）。
func clientIP(c *gin.Context) string {
	// 直连 IP（不信任未经配置的转发头）
	ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return ip
}

// 预定义的限流器实例
var (
	// loginLimiter 登录限流：每 IP 每分钟最多 10 次
	loginLimiter = newRateLimiter(10, time.Minute)
	// registerLimiter 注册限流：每 IP 每小时最多 5 次
	registerLimiter = newRateLimiter(5, time.Hour)
	// sendCodeLimiter 验证码限流：每 IP 每分钟最多 3 次
	sendCodeLimiter = newRateLimiter(3, time.Minute)
	// passwordLimiter 密码重置限流：每 IP 每分钟最多 5 次
	passwordLimiter = newRateLimiter(5, time.Minute)
	// verifyCodeLimiter 验证码校验限流：每 IP 每分钟最多 20 次（防 6 位码爆破）
	verifyCodeLimiter = newRateLimiter(20, time.Minute)
)

// RateLimitLogin 登录限流中间件
func RateLimitLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := clientIP(c)
		if !loginLimiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "登录尝试过于频繁，请1分钟后再试"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RateLimitRegister 注册限流中间件
func RateLimitRegister() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := clientIP(c)
		if !registerLimiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "注册过于频繁，请1小时后再试"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RateLimitSendCode 发送验证码限流中间件
func RateLimitSendCode() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := clientIP(c)
		if !sendCodeLimiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "发送验证码过于频繁，请1分钟后再试"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RateLimitPassword 密码重置限流中间件
func RateLimitPassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := clientIP(c)
		if !passwordLimiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "操作过于频繁，请1分钟后再试"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RateLimitVerifyCode 验证码校验限流中间件
func RateLimitVerifyCode() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := clientIP(c)
		if !verifyCodeLimiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "验证过于频繁，请1分钟后再试"})
			c.Abort()
			return
		}
		c.Next()
	}
}
