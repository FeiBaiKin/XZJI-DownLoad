package handler

import (
	"strconv"

	"xzji/backend/internal/apperr"
	"xzji/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// === 搜索 ===

type searchReq struct {
	Keyword string `json:"keyword"`
}

func PublicSearch(c *gin.Context) {
	var req searchReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Keyword == "" {
		Fail(c, apperr.BadRequest("请提供有效的关键词"))
		return
	}
	res, err := contentSvc(c).SearchKeyword(req.Keyword)
	if err != nil {
		Fail(c, err)
		return
	}
	// 公开搜索脱敏：付费链接的真实 url/code 不下发给客户端，避免右键/F12 绕过付费。
	// 真实地址仅在鉴权的 purchase-link / purchase-status 校验通过后由 ResolveBlock 返回。
	if res != nil {
		res.Response = service.RedactPaidLinks(res.Response)
	}
	OK(c, res)
}

// === 页面 ===

func PublicPage(c *gin.Context) {
	slug := c.Param("slug")
	p, err := contentSvc(c).PublicPage(slug)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, p)
}

func PublicPages(c *gin.Context) {
	list, err := contentSvc(c).PublicPages()
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, list)
}

// === 公告 ===

func PublicAnnouncements(c *gin.Context) {
	list, err := contentSvc(c).ActiveAnnouncements()
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, list)
}

// === 分类 ===

func PublicCategories(c *gin.Context) {
	list, err := contentSvc(c).CategoriesList()
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, list)
}

// === 关键词（公开列表，支持按分类筛选 + 分页） ===

func PublicKeywords(c *gin.Context) {
	var catID *uint
	if v := c.Query("category"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err == nil {
			uid := uint(id)
			catID = &uid
		}
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "12"))
	res, err := contentSvc(c).PublicKeywords(catID, page, pageSize)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, res)
}

// === 精选资源（每分类取前N条拼成精选，整体分页） ===

func PublicFeaturedKeywords(c *gin.Context) {
	perCategory, _ := strconv.Atoi(c.DefaultQuery("per_category", "4"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "12"))
	res, err := contentSvc(c).PublicFeaturedKeywords(perCategory, page, pageSize)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, res)
}
