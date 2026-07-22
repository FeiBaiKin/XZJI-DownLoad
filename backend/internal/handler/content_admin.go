package handler

import (
	"encoding/json"
	"strconv"
	"strings"

	"xzji/backend/internal/apperr"
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
	if err != nil {
		return 0, apperr.BadRequest("无效ID")
	}
	return uint(id), nil
}

// === 关键词 (admin) ===

func AdminKeywordsList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "15"))
	var categoryID *uint
	if v := c.Query("category"); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil && id > 0 {
			cid := uint(id)
			categoryID = &cid
		}
	}
	status := c.Query("status")
	res, err := contentSvc(c).KeywordsListPaged(categoryID, status, page, limit)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, res)
}

func AdminKeywordGet(c *gin.Context) {
	id, err := paramID(c)
	if err != nil {
		Fail(c, err)
		return
	}
	kw, err := contentSvc(c).KeywordGet(id)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, kw)
}

func AdminKeywordCreate(c *gin.Context) {
	var req struct {
		Keyword  string `json:"keyword"`
		Response any    `json:"response"`
		Status   string `json:"status"`
	}
	if !BindJSON(c, &req) {
		return
	}
	kw, err := contentSvc(c).KeywordCreate(service.KeywordCreate{
		Keyword:  req.Keyword,
		Response: jsonString(req.Response),
		Status:   req.Status,
	})
	if err != nil {
		Fail(c, err)
		return
	}
	Created(c, kw)
}

func AdminKeywordUpdate(c *gin.Context) {
	id, err := paramID(c)
	if err != nil {
		Fail(c, err)
		return
	}
	var body map[string]any
	if !BindJSON(c, &body) {
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
	if v, ok := body["status"].(string); ok {
		req.Status = &v
	}
	kw, err := contentSvc(c).KeywordUpdate(id, req)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, kw)
}

func AdminKeywordDelete(c *gin.Context) {
	id, err := paramID(c)
	if err != nil {
		Fail(c, err)
		return
	}
	if err := contentSvc(c).KeywordDelete(id); err != nil {
		Fail(c, err)
		return
	}
	Message(c, "删除成功")
}

func AdminKeywordCategoriesGet(c *gin.Context) {
	id, err := paramID(c)
	if err != nil {
		Fail(c, err)
		return
	}
	ids, err := contentSvc(c).KeywordCategoriesGet(id)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, ids)
}

func AdminKeywordCategoriesSet(c *gin.Context) {
	id, err := paramID(c)
	if err != nil {
		Fail(c, err)
		return
	}
	var ids []uint
	if !BindJSON(c, &ids) {
		return
	}
	if err := contentSvc(c).KeywordCategoriesSet(id, ids); err != nil {
		Fail(c, err)
		return
	}
	Message(c, "设置成功")
}

// AdminKeywordsBatchStatus POST /admin/keywords/batch-status
func AdminKeywordsBatchStatus(c *gin.Context) {
	var req struct {
		IDs    []uint `json:"ids"`
		Status string `json:"status"`
	}
	if !BindJSON(c, &req) {
		return
	}
	n, err := contentSvc(c).BatchSetKeywordStatus(req.IDs, req.Status)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"message": "状态已更新", "updated": n})
}

// AdminKeywordsExport GET /admin/keywords/export?format=json|csv
func AdminKeywordsExport(c *gin.Context) {
	var categoryID *uint
	if v := c.Query("category"); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil && id > 0 {
			cid := uint(id)
			categoryID = &cid
		}
	}
	status := c.Query("status")
	bundle, err := contentSvc(c).ExportKeywords(categoryID, status)
	if err != nil {
		Fail(c, err)
		return
	}
	format := strings.ToLower(c.DefaultQuery("format", "json"))
	if format == "csv" {
		c.Header("Content-Disposition", `attachment; filename="keywords-export.csv"`)
		c.Header("Content-Type", "text/csv; charset=utf-8")
		// UTF-8 BOM，方便 Excel 打开
		if _, err := c.Writer.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
			Fail(c, apperr.Wrap("写出失败", err))
			return
		}
		if err := service.WriteKeywordsCSV(c.Writer, bundle); err != nil {
			// 头已写出，无法再改状态码
			return
		}
		return
	}
	c.Header("Content-Disposition", `attachment; filename="keywords-export.json"`)
	OK(c, bundle)
}

// AdminKeywordsImport POST /admin/keywords/import
// 支持 JSON body {items, mode}，或 multipart: file + mode
func AdminKeywordsImport(c *gin.Context) {
	ct := c.ContentType()
	var req service.KeywordImportRequest

	if strings.Contains(ct, "multipart/form-data") {
		mode := c.PostForm("mode")
		file, err := c.FormFile("file")
		if err != nil {
			Fail(c, apperr.BadRequest("请上传导入文件"))
			return
		}
		f, err := file.Open()
		if err != nil {
			Fail(c, apperr.Wrap("读取文件失败", err))
			return
		}
		defer f.Close()
		name := strings.ToLower(file.Filename)
		if strings.HasSuffix(name, ".csv") {
			items, err := service.ParseKeywordsCSV(f)
			if err != nil {
				Fail(c, err)
				return
			}
			req.Items = items
		} else {
			// 默认按 JSON bundle 或 items 数组
			var raw json.RawMessage
			if err := json.NewDecoder(f).Decode(&raw); err != nil {
				Fail(c, apperr.BadRequest("JSON 文件解析失败"))
				return
			}
			var bundle service.KeywordExportBundle
			if err := json.Unmarshal(raw, &bundle); err == nil && len(bundle.Items) > 0 {
				req.Items = bundle.Items
			} else {
				var items []service.KeywordExportItem
				if err := json.Unmarshal(raw, &items); err != nil {
					Fail(c, apperr.BadRequest("JSON 格式无效，需为导出包或 items 数组"))
					return
				}
				req.Items = items
			}
		}
		req.Mode = mode
	} else {
		if !BindJSON(c, &req) {
			return
		}
	}

	res, err := contentSvc(c).ImportKeywords(req)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, res)
}

// === 分类 (admin) ===

func AdminCategoriesList(c *gin.Context) {
	list, err := contentSvc(c).CategoriesList()
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, list)
}

func AdminCategoryCreate(c *gin.Context) {
	var req service.CategoryCreate
	if !BindJSON(c, &req) {
		return
	}
	m, err := contentSvc(c).CategoryCreate(req)
	if err != nil {
		Fail(c, err)
		return
	}
	Created(c, m)
}

func AdminCategoryUpdate(c *gin.Context) {
	id, err := paramID(c)
	if err != nil {
		Fail(c, err)
		return
	}
	var req service.CategoryUpdate
	if !BindJSON(c, &req) {
		return
	}
	m, err := contentSvc(c).CategoryUpdate(id, req)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, m)
}

func AdminCategoryDelete(c *gin.Context) {
	id, err := paramID(c)
	if err != nil {
		Fail(c, err)
		return
	}
	if err := contentSvc(c).CategoryDelete(id); err != nil {
		Fail(c, err)
		return
	}
	Message(c, "删除成功")
}

// === 页面 (admin) ===

func AdminPagesList(c *gin.Context) {
	list, err := contentSvc(c).PagesList()
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, list)
}

func AdminPageCreate(c *gin.Context) {
	var req service.PageCreate
	if !BindJSON(c, &req) {
		return
	}
	m, err := contentSvc(c).PageCreate(req)
	if err != nil {
		Fail(c, err)
		return
	}
	Created(c, m)
}

func AdminPageUpdate(c *gin.Context) {
	id, err := paramID(c)
	if err != nil {
		Fail(c, err)
		return
	}
	var req service.PageUpdate
	if !BindJSON(c, &req) {
		return
	}
	m, err := contentSvc(c).PageUpdate(id, req)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, m)
}

func AdminPageDelete(c *gin.Context) {
	id, err := paramID(c)
	if err != nil {
		Fail(c, err)
		return
	}
	if err := contentSvc(c).PageDelete(id); err != nil {
		Fail(c, err)
		return
	}
	Message(c, "删除成功")
}

// === 公告 (admin) ===

func AdminAnnouncementsList(c *gin.Context) {
	list, err := contentSvc(c).AnnouncementsList()
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, list)
}

func AdminAnnouncementCreate(c *gin.Context) {
	var req service.AnnouncementCreate
	if !BindJSON(c, &req) {
		return
	}
	m, err := contentSvc(c).AnnouncementCreate(req)
	if err != nil {
		Fail(c, err)
		return
	}
	Created(c, m)
}

func AdminAnnouncementUpdate(c *gin.Context) {
	id, err := paramID(c)
	if err != nil {
		Fail(c, err)
		return
	}
	var req service.AnnouncementUpdate
	if !BindJSON(c, &req) {
		return
	}
	m, err := contentSvc(c).AnnouncementUpdate(id, req)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, m)
}

func AdminAnnouncementDelete(c *gin.Context) {
	id, err := paramID(c)
	if err != nil {
		Fail(c, err)
		return
	}
	if err := contentSvc(c).AnnouncementDelete(id); err != nil {
		Fail(c, err)
		return
	}
	Message(c, "删除成功")
}
