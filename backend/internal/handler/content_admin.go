package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"xzji/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// contentSvc 从 context 获取 ContentService（依赖注入）
func contentSvc(c *gin.Context) *service.ContentService {
	return getApp(c).ContentSvc
}

func jsonString(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func paramID(c *gin.Context) (uint, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	return uint(id), err
}

// === 关键词 (admin) ===

func AdminKeywordsList(c *gin.Context) {
	kws, err := contentSvc(c).KeywordsList()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取关键词失败"})
		return
	}
	c.JSON(http.StatusOK, kws)
}

func AdminKeywordGet(c *gin.Context) {
	id, err := paramID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效ID"})
		return
	}
	kw, err := contentSvc(c).KeywordGet(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "关键词不存在"})
		return
	}
	c.JSON(http.StatusOK, kw)
}

func AdminKeywordCreate(c *gin.Context) {
	var req struct {
		Keyword  string `json:"keyword"`
		Response any    `json:"response"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}
	kw, err := contentSvc(c).KeywordCreate(service.KeywordCreate{Keyword: req.Keyword, Response: jsonString(req.Response)})
	if err != nil {
		code := http.StatusBadRequest
		if err.Error() == "关键词已存在" {
			code = http.StatusConflict
		}
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, kw)
}

func AdminKeywordUpdate(c *gin.Context) {
	id, err := paramID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效ID"})
		return
	}
	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}
	req := service.KeywordUpdate{}
	if v, ok := body["keyword"].(string); ok {
		req.Keyword = &v
	}
	if v, ok := body["response"]; ok {
		s := jsonString(v)
		req.Response = &s
	}
	if v, ok := body["icon"].(string); ok {
		req.Icon = &v
	}
if v, ok := body["attributes"].(string); ok {
			req.Attributes = &v
		}
	kw, err := contentSvc(c).KeywordUpdate(id, req)
	if err != nil {
		code := http.StatusBadRequest
		if err.Error() == "关键词已存在" {
			code = http.StatusConflict
		}
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, kw)
}

func AdminKeywordDelete(c *gin.Context) {
	id, err := paramID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效ID"})
		return
	}
	if err := contentSvc(c).KeywordDelete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func AdminKeywordCategoriesGet(c *gin.Context) {
	id, _ := paramID(c)
	ids, err := contentSvc(c).KeywordCategoriesGet(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}
	c.JSON(http.StatusOK, ids)
}

func AdminKeywordCategoriesSet(c *gin.Context) {
	id, _ := paramID(c)
	var ids []uint
	if err := c.ShouldBindJSON(&ids); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := contentSvc(c).KeywordCategoriesSet(id, ids); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "设置失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "设置成功"})
}

// === 分类 (admin) ===

func AdminCategoriesList(c *gin.Context) {
	list, err := contentSvc(c).CategoriesList()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取分类失败"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func AdminCategoryCreate(c *gin.Context) {
	var req service.CategoryCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}
	m, err := contentSvc(c).CategoryCreate(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, m)
}

func AdminCategoryUpdate(c *gin.Context) {
	id, _ := paramID(c)
	var req service.CategoryUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}
	m, err := contentSvc(c).CategoryUpdate(id, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, m)
}

func AdminCategoryDelete(c *gin.Context) {
	id, _ := paramID(c)
	if err := contentSvc(c).CategoryDelete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// === 页面 (admin) ===

func AdminPagesList(c *gin.Context) {
	list, err := contentSvc(c).PagesList()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取页面失败"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func AdminPageCreate(c *gin.Context) {
	var req service.PageCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}
	m, err := contentSvc(c).PageCreate(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, m)
}

func AdminPageUpdate(c *gin.Context) {
	id, _ := paramID(c)
	var req service.PageUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}
	m, err := contentSvc(c).PageUpdate(id, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, m)
}

func AdminPageDelete(c *gin.Context) {
	id, _ := paramID(c)
	if err := contentSvc(c).PageDelete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// === 公告 (admin) ===

func AdminAnnouncementsList(c *gin.Context) {
	list, err := contentSvc(c).AnnouncementsList()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取公告失败"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func AdminAnnouncementCreate(c *gin.Context) {
	var req service.AnnouncementCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}
	m, err := contentSvc(c).AnnouncementCreate(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, m)
}

func AdminAnnouncementUpdate(c *gin.Context) {
	id, _ := paramID(c)
	var req service.AnnouncementUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}
	m, err := contentSvc(c).AnnouncementUpdate(id, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, m)
}

func AdminAnnouncementDelete(c *gin.Context) {
	id, _ := paramID(c)
	if err := contentSvc(c).AnnouncementDelete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
