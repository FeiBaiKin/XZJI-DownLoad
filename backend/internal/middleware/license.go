package middleware

import (
	"net/http"

	"xzji/backend/internal/license"

	"github.com/gin-gonic/gin"
)

// LicenseGuard 授权停服中间件：授权无效时拦截前台业务接口，
// 返回统一文案与 license_blocked 错误码（前端据此显示全页停服遮罩）。
// 放行范围（登录、管理后台、站点设置、支付回调、静态资源）由路由装配决定，
// 本中间件只做判定，见 router.SetupFull。
func LicenseGuard(m *license.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if m == nil || m.Valid() {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": license.BlockedMessage,
			"code":  "license_blocked",
		})
	}
}
