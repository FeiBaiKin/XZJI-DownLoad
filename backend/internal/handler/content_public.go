package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"xzji/backend/internal/service"
)

// === 搜索 ===

type searchReq struct {
	Keyword string `json:"keyword"`
}

func PublicSearch(c *gin.Context) {
	var req searchReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供有效的关键词"})
		return
	}
	res, _ := contentSvc(c).SearchKeyword(req.Keyword)
	// 公开搜索脱敏：付费链接的真实 url/code 不下发给客户端，避免右键/F12 绕过付费。
	// 真实地址仅在鉴权的 purchase-link / purchase-status 校验通过后由 ResolveBlock 返回。
	if res != nil {
		res.Response = service.RedactPaidLinks(res.Response)
	}
	c.JSON(http.StatusOK, res)
}

// === 页面 ===

func PublicPage(c *gin.Context) {
	slug := c.Param("slug")
	p, err := contentSvc(c).PublicPage(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "页面不存在"})
		return
	}
	c.JSON(http.StatusOK, p)
}

func PublicPages(c *gin.Context) {
	list, _ := contentSvc(c).PublicPages()
	c.JSON(http.StatusOK, list)
}

// === 公告 ===

func PublicAnnouncements(c *gin.Context) {
	list, _ := contentSvc(c).ActiveAnnouncements()
	c.JSON(http.StatusOK, list)
}

// === 分类 ===

func PublicCategories(c *gin.Context) {
	list, _ := contentSvc(c).CategoriesList()
	c.JSON(http.StatusOK, list)
}

// === 关键词（公开列表，支持按分类筛选） ===

func PublicKeywords(c *gin.Context) {
	var catID *uint
	if v := c.Query("category"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err == nil {
			uid := uint(id)
			catID = &uid
		}
	}
	list, _ := contentSvc(c).PublicKeywords(catID)
	c.JSON(http.StatusOK, list)
}
