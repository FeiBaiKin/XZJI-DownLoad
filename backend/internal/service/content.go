package service

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"xzji/backend/internal/apperr"
	"xzji/backend/internal/models"
)

type ContentService struct{ DB *gorm.DB }

// ====== 关键词 ======

type KeywordCreate struct {
	Keyword  string `json:"keyword"`
	Response string `json:"response"` // JSON 字符串或纯文本，handler 层已统一为 JSON 字符串
	Status   string `json:"status"`   // draft | published，默认 published
}

type KeywordUpdate struct {
	Keyword    *string `json:"keyword"`
	Response   *string `json:"response"`
	Icon       *string `json:"icon"`
	Attributes *string `json:"attributes"`
	Status     *string `json:"status"`
}

// NormalizeKeywordStatus 校验并归一化状态；空串按 published。
func NormalizeKeywordStatus(s string) (string, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return models.KeywordStatusPublished, nil
	}
	if s != models.KeywordStatusDraft && s != models.KeywordStatusPublished {
		return "", apperr.BadRequest("状态仅支持 draft 或 published")
	}
	return s, nil
}

func (s *ContentService) KeywordsList() ([]models.KeywordResponse, error) {
	var list []models.KeywordResponse
	if err := s.DB.Order("id ASC").Find(&list).Error; err != nil {
		return nil, apperr.Wrap("获取关键词失败", err)
	}
	return list, nil
}

// AdminKeywordItem 后台关键词列表项：在 KeywordResponse 基础上附带所属分类 ID
type AdminKeywordItem struct {
	models.KeywordResponse
	CategoryIDs []uint `json:"category_ids"`
}

// PaginatedAdminKeywords 后台关键词分页结果
type PaginatedAdminKeywords struct {
	Items []AdminKeywordItem `json:"items"`
	Total int64              `json:"total"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
}

// KeywordsListPaged 后台关键词分页列表（支持按分类/状态筛选），批量装配 category_ids 避免 N+1
func (s *ContentService) KeywordsListPaged(categoryID *uint, status string, page, limit int) (*PaginatedAdminKeywords, error) {
	page, limit = normalizePage(page, limit, 100)
	q := s.DB.Model(&models.KeywordResponse{})
	if categoryID != nil && *categoryID > 0 {
		q = q.Where("id IN (?)",
			s.DB.Table("keyword_category_links").Select("keyword_id").Where("category_id = ?", *categoryID))
	}
	if st, err := NormalizeKeywordStatus(status); err == nil && status != "" {
		q = q.Where("status = ?", st)
	} else if status != "" {
		return nil, err
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, apperr.Wrap("获取关键词失败", err)
	}
	var rows []models.KeywordResponse
	if err := q.Order("id ASC").Offset((page - 1) * limit).Limit(limit).Find(&rows).Error; err != nil {
		return nil, apperr.Wrap("获取关键词失败", err)
	}

	// 批量查询本页关键词的分类关联
	ids := make([]uint, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	linkMap := make(map[uint][]uint, len(rows))
	if len(ids) > 0 {
		var links []models.KeywordCategoryLink
		if err := s.DB.Where("keyword_id IN ?", ids).Find(&links).Error; err != nil {
			return nil, apperr.Wrap("获取关键词分类失败", err)
		}
		for _, l := range links {
			linkMap[l.KeywordID] = append(linkMap[l.KeywordID], l.CategoryID)
		}
	}

	items := make([]AdminKeywordItem, len(rows))
	for i, r := range rows {
		cids := linkMap[r.ID]
		if cids == nil {
			cids = []uint{}
		}
		if r.Status == "" {
			r.Status = models.KeywordStatusPublished
		}
		items[i] = AdminKeywordItem{KeywordResponse: r, CategoryIDs: cids}
	}
	return &PaginatedAdminKeywords{Items: items, Total: total, Page: page, Limit: limit}, nil
}

func (s *ContentService) KeywordGet(id uint) (*models.KeywordResponse, error) {
	var kw models.KeywordResponse
	if err := s.DB.First(&kw, id).Error; err != nil {
		return nil, apperr.NotFound("关键词不存在")
	}
	if kw.Status == "" {
		kw.Status = models.KeywordStatusPublished
	}
	return &kw, nil
}

func (s *ContentService) KeywordCreate(req KeywordCreate) (*models.KeywordResponse, error) {
	kw := strings.TrimSpace(strings.ToLower(req.Keyword))
	if kw == "" {
		return nil, apperr.BadRequest("关键词不能为空")
	}
	status, err := NormalizeKeywordStatus(req.Status)
	if err != nil {
		return nil, err
	}
	var exist int64
	s.DB.Model(&models.KeywordResponse{}).Where("keyword = ?", kw).Count(&exist)
	if exist > 0 {
		return nil, apperr.Conflict("关键词已存在")
	}
	m := models.KeywordResponse{Keyword: kw, Response: req.Response, Status: status}
	if err := s.DB.Create(&m).Error; err != nil {
		return nil, apperr.Wrap("创建失败", err)
	}
	return &m, nil
}

func (s *ContentService) KeywordUpdate(id uint, req KeywordUpdate) (*models.KeywordResponse, error) {
	var m models.KeywordResponse
	if err := s.DB.First(&m, id).Error; err != nil {
		return nil, apperr.NotFound("关键词不存在")
	}
	if req.Keyword != nil {
		kw := strings.TrimSpace(strings.ToLower(*req.Keyword))
		if kw != m.Keyword {
			var cnt int64
			s.DB.Model(&models.KeywordResponse{}).Where("keyword = ? AND id != ?", kw, id).Count(&cnt)
			if cnt > 0 {
				return nil, apperr.Conflict("关键词已存在")
			}
			m.Keyword = kw
		}
	}
	if req.Response != nil {
		m.Response = *req.Response
	}
	if req.Icon != nil {
		m.Icon = *req.Icon
	}
	if req.Attributes != nil {
		m.Attributes = *req.Attributes
	}
	if req.Status != nil {
		st, err := NormalizeKeywordStatus(*req.Status)
		if err != nil {
			return nil, err
		}
		m.Status = st
	}
	if err := s.DB.Save(&m).Error; err != nil {
		return nil, apperr.Wrap("更新失败", err)
	}
	return &m, nil
}

func (s *ContentService) KeywordDelete(id uint) error {
	// 事务内删除关键词及其关联记录，防止孤儿数据
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("keyword_id = ?", id).Delete(&models.KeywordCategoryLink{}).Error; err != nil {
			return apperr.Wrap("删除关联失败", err)
		}
		res := tx.Delete(&models.KeywordResponse{}, id)
		if res.Error != nil {
			return apperr.Wrap("删除失败", res.Error)
		}
		if res.RowsAffected == 0 {
			return apperr.NotFound("关键词不存在")
		}
		return nil
	})
}

// BatchSetKeywordStatus 批量上下架
func (s *ContentService) BatchSetKeywordStatus(ids []uint, status string) (int64, error) {
	st, err := NormalizeKeywordStatus(status)
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, apperr.BadRequest("请选择关键词")
	}
	if len(ids) > 500 {
		return 0, apperr.BadRequest("单次最多操作 500 条")
	}
	res := s.DB.Model(&models.KeywordResponse{}).Where("id IN ?", ids).Update("status", st)
	if res.Error != nil {
		return 0, apperr.Wrap("更新状态失败", res.Error)
	}
	return res.RowsAffected, nil
}

// ====== 关键词←→分类关联 ======

func (s *ContentService) KeywordCategoriesGet(keywordID uint) ([]uint, error) {
	var links []models.KeywordCategoryLink
	if err := s.DB.Where("keyword_id = ?", keywordID).Find(&links).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, len(links))
	for i, l := range links {
		ids[i] = l.CategoryID
	}
	return ids, nil
}

func (s *ContentService) KeywordCategoriesSet(keywordID uint, categoryIDs []uint) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		tx.Where("keyword_id = ?", keywordID).Delete(&models.KeywordCategoryLink{})
		for _, cid := range categoryIDs {
			link := models.KeywordCategoryLink{KeywordID: keywordID, CategoryID: cid}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&link).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ====== 分类 ======

type CategoryCreate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

type CategoryUpdate struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	SortOrder   *int    `json:"sort_order"`
}

func (s *ContentService) CategoriesList() ([]models.KeywordCategory, error) {
	var list []models.KeywordCategory
	if err := s.DB.Order("sort_order ASC, id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (s *ContentService) CategoryCreate(req CategoryCreate) (*models.KeywordCategory, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, apperr.BadRequest("分类名称不能为空")
	}
	m := models.KeywordCategory{Name: req.Name, Description: req.Description, SortOrder: req.SortOrder}
	if err := s.DB.Create(&m).Error; err != nil {
		return nil, apperr.Wrap("创建失败", err)
	}
	return &m, nil
}

func (s *ContentService) CategoryUpdate(id uint, req CategoryUpdate) (*models.KeywordCategory, error) {
	var m models.KeywordCategory
	if err := s.DB.First(&m, id).Error; err != nil {
		return nil, apperr.NotFound("分类不存在")
	}
	if req.Name != nil {
		m.Name = *req.Name
	}
	if req.Description != nil {
		m.Description = *req.Description
	}
	if req.SortOrder != nil {
		m.SortOrder = *req.SortOrder
	}
	if err := s.DB.Save(&m).Error; err != nil {
		return nil, apperr.Wrap("更新失败", err)
	}
	return &m, nil
}

func (s *ContentService) CategoryDelete(id uint) error {
	// 事务内删除分类及其关联记录，防止孤儿数据
	return s.DB.Transaction(func(tx *gorm.DB) error {
		// 删除关键词-分类关联
		if err := tx.Where("category_id = ?", id).Delete(&models.KeywordCategoryLink{}).Error; err != nil {
			return fmt.Errorf("删除关联失败: %w", err)
		}
		// 删除分类
		if err := tx.Delete(&models.KeywordCategory{}, id).Error; err != nil {
			return fmt.Errorf("删除失败: %w", err)
		}
		return nil
	})
}

// ====== 页面 ======

type PageCreate struct {
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	Content     string `json:"content"`
	IsPublished bool   `json:"is_published"`
}

type PageUpdate struct {
	Title       *string `json:"title"`
	Slug        *string `json:"slug"`
	Content     *string `json:"content"`
	IsPublished *bool   `json:"is_published"`
}

func (s *ContentService) PagesList() ([]models.Page, error) {
	var list []models.Page
	if err := s.DB.Order("id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (s *ContentService) PageCreate(req PageCreate) (*models.Page, error) {
	m := models.Page{Title: req.Title, Slug: req.Slug, Content: req.Content, IsPublished: req.IsPublished}
	if err := s.DB.Create(&m).Error; err != nil {
		return nil, fmt.Errorf("创建失败")
	}
	return &m, nil
}

func (s *ContentService) PageUpdate(id uint, req PageUpdate) (*models.Page, error) {
	var m models.Page
	if err := s.DB.First(&m, id).Error; err != nil {
		return nil, fmt.Errorf("页面不存在")
	}
	if req.Title != nil {
		m.Title = *req.Title
	}
	if req.Slug != nil {
		m.Slug = *req.Slug
	}
	if req.Content != nil {
		m.Content = *req.Content
	}
	if req.IsPublished != nil {
		m.IsPublished = *req.IsPublished
	}
	if err := s.DB.Save(&m).Error; err != nil {
		return nil, fmt.Errorf("更新失败")
	}
	return &m, nil
}

func (s *ContentService) PageDelete(id uint) error {
	if err := s.DB.Delete(&models.Page{}, id).Error; err != nil {
		return fmt.Errorf("删除失败")
	}
	return nil
}

// ====== 公告 ======

type AnnouncementCreate struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	Type     string `json:"type"`
	IsActive bool   `json:"is_active"`
}

type AnnouncementUpdate struct {
	Title    *string `json:"title"`
	Content  *string `json:"content"`
	Type     *string `json:"type"`
	IsActive *bool   `json:"is_active"`
}

func (s *ContentService) AnnouncementsList() ([]models.Announcement, error) {
	var list []models.Announcement
	if err := s.DB.Order("id DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (s *ContentService) AnnouncementCreate(req AnnouncementCreate) (*models.Announcement, error) {
	m := models.Announcement{Title: req.Title, Content: req.Content, Type: req.Type, IsActive: req.IsActive}
	if err := s.DB.Create(&m).Error; err != nil {
		return nil, fmt.Errorf("创建失败")
	}
	return &m, nil
}

func (s *ContentService) AnnouncementUpdate(id uint, req AnnouncementUpdate) (*models.Announcement, error) {
	var m models.Announcement
	if err := s.DB.First(&m, id).Error; err != nil {
		return nil, fmt.Errorf("公告不存在")
	}
	if req.Title != nil {
		m.Title = *req.Title
	}
	if req.Content != nil {
		m.Content = *req.Content
	}
	if req.Type != nil {
		m.Type = *req.Type
	}
	if req.IsActive != nil {
		m.IsActive = *req.IsActive
	}
	if err := s.DB.Save(&m).Error; err != nil {
		return nil, fmt.Errorf("更新失败")
	}
	return &m, nil
}

func (s *ContentService) AnnouncementDelete(id uint) error {
	return s.DB.Delete(&models.Announcement{}, id).Error
}

// ====== 公开端点辅助 ======

// PublicKeyword 公开关键词预览（含创建/更新时间，便于前台"最近更新"展示）
type PublicKeyword struct {
	ID          uint      `json:"id"`
	Keyword     string    `json:"keyword"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	Attributes  string    `json:"attributes"` // JSON 字符串: [{label,value}]，前端解析后渲染
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PaginatedKeywords 公开关键词分页结果
type PaginatedKeywords struct {
	Items    []PublicKeyword `json:"items"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

// toPublicKeyword 把 KeywordResponse 行转换为公开预览（描述截断 120 字符，附时间戳）
func toPublicKeyword(r models.KeywordResponse) PublicKeyword {
	preview := extractDescription(r.Response)
	if len([]rune(preview)) > 120 {
		preview = string([]rune(preview)[:120]) + "..."
	}
	return PublicKeyword{
		ID:          r.ID,
		Keyword:     r.Keyword,
		Description: preview,
		Icon:        r.Icon,
		Attributes:  r.Attributes,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// normalizePage 规范化分页参数，返回 (page, pageSize)
func normalizePage(page, pageSize, maxPageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 12
	}
	if maxPageSize > 0 && pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

// publishedScope 前台仅展示已上架资源（status 空串兼容旧数据视为 published）
func publishedScope(db *gorm.DB) *gorm.DB {
	return db.Where("(status = ? OR status = '' OR status IS NULL)", models.KeywordStatusPublished)
}

// PublicKeywords 公开关键词列表（支持按分类筛选 + 分页），仅已上架
func (s *ContentService) PublicKeywords(categoryID *uint, page, pageSize int) (*PaginatedKeywords, error) {
	page, pageSize = normalizePage(page, pageSize, 60)
	q := publishedScope(s.DB.Model(&models.KeywordResponse{}))
	if categoryID != nil && *categoryID > 0 {
		q = q.Where("id IN (?)",
			s.DB.Table("keyword_category_links").Select("keyword_id").Where("category_id = ?", *categoryID))
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, apperr.Wrap("获取关键词失败", err)
	}
	var rows []models.KeywordResponse
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, apperr.Wrap("获取关键词失败", err)
	}
	items := make([]PublicKeyword, len(rows))
	for i, r := range rows {
		items[i] = toPublicKeyword(r)
	}
	return &PaginatedKeywords{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// PublicFeaturedKeywords 精选资源：每个分类按 id DESC 取前 perCategory 条，flatten 后整体分页。
// 仅已上架。精选总量有限（分类数 × perCategory），采用内存分页可接受。
func (s *ContentService) PublicFeaturedKeywords(perCategory, page, pageSize int) (*PaginatedKeywords, error) {
	if perCategory < 1 {
		perCategory = 4
	}
	page, pageSize = normalizePage(page, pageSize, 60)

	cats, err := s.CategoriesList()
	if err != nil {
		return nil, err
	}
	var all []PublicKeyword
	for _, cat := range cats {
		var rows []models.KeywordResponse
		if err := publishedScope(s.DB).
			Where("id IN (?)", s.DB.Table("keyword_category_links").
				Select("keyword_id").Where("category_id = ?", cat.ID)).
			Order("id DESC").Limit(perCategory).Find(&rows).Error; err != nil {
			return nil, apperr.Wrap("获取精选资源失败", err)
		}
		for _, r := range rows {
			all = append(all, toPublicKeyword(r))
		}
	}
	total := int64(len(all))
	start := (page - 1) * pageSize
	if start > len(all) {
		start = len(all)
	}
	end := start + pageSize
	if end > len(all) {
		end = len(all)
	}
	items := all[start:end]
	if items == nil {
		items = []PublicKeyword{}
	}
	return &PaginatedKeywords{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// PublicPage 按 slug 获取已发布页面
func (s *ContentService) PublicPage(slug string) (*models.Page, error) {
	var p models.Page
	if err := s.DB.Where("slug = ? AND is_published = ?", slug, true).First(&p).Error; err != nil {
		return nil, apperr.NotFound("页面不存在")
	}
	return &p, nil
}

// PublicPages 已发布页面列表
func (s *ContentService) PublicPages() ([]models.Page, error) {
	var list []models.Page
	if err := s.DB.Where("is_published = ?", true).Order("id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// ActiveAnnouncements 活跃公告列表
func (s *ContentService) ActiveAnnouncements() ([]models.Announcement, error) {
	var list []models.Announcement
	if err := s.DB.Where("is_active = ?", true).Order("id DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// SearchKeyword 搜索关键词并返回 response
type SearchResult struct {
	Keyword    string    `json:"keyword"`
	Response   string    `json:"response"` // JSON 字符串
	Icon       string    `json:"icon,omitempty"`
	Attributes string    `json:"attributes,omitempty"` // JSON 字符串: [{label,value}]
	Timestamp  time.Time `json:"timestamp"`
	Source     string    `json:"source"`
	KeywordID  *uint     `json:"keywordId,omitempty"`
}

var fallbackResponses = map[string]string{
	"hello":      `"你好！欢迎使用关键词搜索应用。"`,
	"node":       `"Node.js 是一个基于 Chrome V8 引擎的 JavaScript 运行时。"`,
	"react":      `"React 是一个用于构建用户界面的 JavaScript 库。"`,
	"postgresql": `"PostgreSQL 是功能强大的开源对象关系型数据库系统。"`,
	"search":     `"搜索功能已启用，正在探索关键词的奥秘。"`,
}

func (s *ContentService) SearchKeyword(keyword string) (*SearchResult, error) {
	kw := strings.TrimSpace(strings.ToLower(keyword))
	if kw == "" {
		return nil, apperr.BadRequest("请提供有效的关键词")
	}
	var m models.KeywordResponse
	// 仅命中已上架资源；草稿/下架对前台等同于未找到
	err := publishedScope(s.DB).Where("keyword = ?", kw).First(&m).Error
	if err == nil {
		kid := m.ID
		return &SearchResult{
			Keyword:    keyword,
			Response:   m.Response,
			Icon:       m.Icon,
			Attributes: m.Attributes,
			Timestamp:  time.Now(),
			Source:     "database",
			KeywordID:  &kid,
		}, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, apperr.Wrap("搜索失败", err)
	}
	// 内存兜底
	if fb, ok := fallbackResponses[kw]; ok {
		return &SearchResult{
			Keyword:   keyword,
			Response:  fb,
			Timestamp: time.Now(),
			Source:    "fallback",
		}, nil
	}
	return &SearchResult{
		Keyword:   keyword,
		Response:  fmt.Sprintf(`"未找到关键词 "%s" 的特定响应，这是一个默认回复。"`, keyword),
		Timestamp: time.Now(),
		Source:    "fallback",
	}, nil
}

// extractDescription 从 JSON response 中提取文本预览
func extractDescription(raw string) string {
	var parsed any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return raw
	}
	switch v := parsed.(type) {
	case []any:
		for _, block := range v {
			if bm, ok := block.(map[string]any); ok {
				if t, _ := bm["type"].(string); t == "text" {
					if txt, _ := bm["text"].(string); txt != "" {
						return txt
					}
				}
				if c, _ := bm["content"].(string); c != "" {
					return c
				}
			}
		}
	case string:
		return v
	}
	return raw
}

// blockAccess 前端对应实现: frontend/src/utils/blockAccess.ts，修改任一侧务必同步另一侧
func blockAccess(b map[string]any) string {
	if a, _ := b["access"].(string); a == "balance" || a == "membership" {
		return a
	}
	if c, ok := toFloat64JSON(b["cost"]); ok && c > 0 {
		return "balance"
	}
	return "free"
}

type LinkBlockDetails struct {
	URL    string
	Code   string
	Access string
	Cost   float64
}

func toFloat64JSON(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}

// RedactPaidLinks 对公开搜索响应中的付费链接块脱敏：清空 url 与 code，
// 保留 label/cost/access/type 以便前端展示价格与访问类型。
// 真实 url 仅在鉴权的 purchase-link / purchase-status 校验通过后由 ResolveBlock 返回，
// 从而避免右键“在新标签页打开”/F12 直接拿到真实地址绕过付费。
func RedactPaidLinks(response string) string {
	var blocks []any
	if err := json.Unmarshal([]byte(response), &blocks); err != nil {
		return response // 非数组（纯文本/JSON 字符串）原样返回
	}
	changed := false
	for _, raw := range blocks {
		b, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if t, _ := b["type"].(string); t != "link" {
			continue
		}
		if blockAccess(b) == "free" {
			continue
		}
		b["url"] = ""
		b["code"] = ""
		changed = true
	}
	if !changed {
		return response // 无付费链接，保持原始格式
	}
	out, err := json.Marshal(blocks)
	if err != nil {
		return response
	}
	return string(out)
}

func ResolveLinkBlock(db *gorm.DB, keywordID uint, blockIndex int) (*LinkBlockDetails, error) {
	var m models.KeywordResponse
	// 用户侧解析仅允许已上架资源，与搜索/列表 publishedScope 对齐
	if e := publishedScope(db).First(&m, keywordID).Error; e != nil {
		return nil, fmt.Errorf("关键词不存在或未上架")
	}
	var blocks []any
	if e := json.Unmarshal([]byte(m.Response), &blocks); e != nil {
		return nil, fmt.Errorf("响应解析失败")
	}
	if blockIndex < 0 || blockIndex >= len(blocks) {
		return nil, fmt.Errorf("内容块不存在")
	}
	b, ok := blocks[blockIndex].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("内容块格式无效")
	}
	if t, _ := b["type"].(string); t != "link" {
		return nil, fmt.Errorf("内容块不是链接")
	}
	url, _ := b["url"].(string)
	code, _ := b["code"].(string)
	cost, _ := toFloat64JSON(b["cost"])
	return &LinkBlockDetails{URL: url, Code: code, Access: blockAccess(b), Cost: cost}, nil
}

// ResolveBlock 取某关键词 response 中指定 blockIndex 的真实 url 与提取码。
// 仅供鉴权后（purchase-link / purchase-status）调用，用于在校验通过后下发真实地址。
func ResolveBlock(db *gorm.DB, keywordID uint, blockIndex int) (url, code string, err error) {
	block, err := ResolveLinkBlock(db, keywordID, blockIndex)
	if err != nil {
		return "", "", err
	}
	return block.URL, block.Code, nil
}

// ====== 关键词批量导入 / 导出 ======

// KeywordExportItem 导出/导入的单条记录（与 DB 解耦，便于跨环境迁移）
type KeywordExportItem struct {
	Keyword        string   `json:"keyword"`
	Response       string   `json:"response"`
	Icon           string   `json:"icon,omitempty"`
	Attributes     string   `json:"attributes,omitempty"`
	Status         string   `json:"status,omitempty"`
	CategoryNames  []string `json:"category_names,omitempty"`
	CategoryIDs    []uint   `json:"category_ids,omitempty"` // 导入时可选；优先 names
}

// KeywordExportBundle JSON 导出包
type KeywordExportBundle struct {
	Version    int                 `json:"version"`
	ExportedAt time.Time           `json:"exported_at"`
	Items      []KeywordExportItem `json:"items"`
}

// KeywordImportResult 导入结果
type KeywordImportResult struct {
	Created  int      `json:"created"`
	Updated  int      `json:"updated"`
	Skipped  int      `json:"skipped"`
	Failed   int      `json:"failed"`
	Errors   []string `json:"errors,omitempty"`
	Total    int      `json:"total"`
}

// ExportKeywords 导出全部关键词（可按状态/分类过滤）
func (s *ContentService) ExportKeywords(categoryID *uint, status string) (*KeywordExportBundle, error) {
	q := s.DB.Model(&models.KeywordResponse{})
	if categoryID != nil && *categoryID > 0 {
		q = q.Where("id IN (?)",
			s.DB.Table("keyword_category_links").Select("keyword_id").Where("category_id = ?", *categoryID))
	}
	if status != "" {
		st, err := NormalizeKeywordStatus(status)
		if err != nil {
			return nil, err
		}
		q = q.Where("status = ?", st)
	}
	var rows []models.KeywordResponse
	if err := q.Order("id ASC").Find(&rows).Error; err != nil {
		return nil, apperr.Wrap("导出失败", err)
	}

	// 分类 id→name
	var cats []models.KeywordCategory
	_ = s.DB.Find(&cats).Error
	catName := map[uint]string{}
	for _, c := range cats {
		catName[c.ID] = c.Name
	}

	ids := make([]uint, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	linkMap := map[uint][]uint{}
	if len(ids) > 0 {
		var links []models.KeywordCategoryLink
		if err := s.DB.Where("keyword_id IN ?", ids).Find(&links).Error; err != nil {
			return nil, apperr.Wrap("导出分类关联失败", err)
		}
		for _, l := range links {
			linkMap[l.KeywordID] = append(linkMap[l.KeywordID], l.CategoryID)
		}
	}

	items := make([]KeywordExportItem, 0, len(rows))
	for _, r := range rows {
		st := r.Status
		if st == "" {
			st = models.KeywordStatusPublished
		}
		names := []string{}
		for _, cid := range linkMap[r.ID] {
			if n, ok := catName[cid]; ok && n != "" {
				names = append(names, n)
			}
		}
		items = append(items, KeywordExportItem{
			Keyword:       r.Keyword,
			Response:      r.Response,
			Icon:          r.Icon,
			Attributes:    r.Attributes,
			Status:        st,
			CategoryNames: names,
			CategoryIDs:   linkMap[r.ID],
		})
	}
	return &KeywordExportBundle{
		Version:    1,
		ExportedAt: time.Now().UTC(),
		Items:      items,
	}, nil
}

// WriteKeywordsCSV 将导出包写成 CSV（列：keyword,status,icon,attributes,categories,response）
func WriteKeywordsCSV(w io.Writer, bundle *KeywordExportBundle) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"keyword", "status", "icon", "attributes", "categories", "response"}); err != nil {
		return err
	}
	for _, it := range bundle.Items {
		st := it.Status
		if st == "" {
			st = models.KeywordStatusPublished
		}
		if err := cw.Write([]string{
			it.Keyword,
			st,
			it.Icon,
			it.Attributes,
			strings.Join(it.CategoryNames, "|"),
			it.Response,
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// KeywordImportRequest 导入请求
type KeywordImportRequest struct {
	Items []KeywordExportItem `json:"items"`
	// Mode: skip（默认，跳过已存在）| overwrite（覆盖内容与状态）
	Mode string `json:"mode"`
}

// ImportKeywords 批量导入关键词
func (s *ContentService) ImportKeywords(req KeywordImportRequest) (*KeywordImportResult, error) {
	mode := strings.TrimSpace(strings.ToLower(req.Mode))
	if mode == "" {
		mode = "skip"
	}
	if mode != "skip" && mode != "overwrite" {
		return nil, apperr.BadRequest("导入模式仅支持 skip 或 overwrite")
	}
	if len(req.Items) == 0 {
		return nil, apperr.BadRequest("导入列表为空")
	}
	if len(req.Items) > 2000 {
		return nil, apperr.BadRequest("单次最多导入 2000 条")
	}

	// 预加载分类名 → id
	var cats []models.KeywordCategory
	_ = s.DB.Find(&cats).Error
	catByName := map[string]uint{}
	for _, c := range cats {
		catByName[strings.ToLower(strings.TrimSpace(c.Name))] = c.ID
	}

	result := &KeywordImportResult{Total: len(req.Items)}
	const maxErrors = 30

	for i, raw := range req.Items {
		kw := strings.TrimSpace(strings.ToLower(raw.Keyword))
		if kw == "" {
			result.Failed++
			if len(result.Errors) < maxErrors {
				result.Errors = append(result.Errors, fmt.Sprintf("第 %d 条：关键词为空", i+1))
			}
			continue
		}
		status, err := NormalizeKeywordStatus(raw.Status)
		if err != nil {
			result.Failed++
			if len(result.Errors) < maxErrors {
				result.Errors = append(result.Errors, fmt.Sprintf("第 %d 条「%s」：%s", i+1, kw, err.Error()))
			}
			continue
		}
		if strings.TrimSpace(raw.Response) == "" {
			result.Failed++
			if len(result.Errors) < maxErrors {
				result.Errors = append(result.Errors, fmt.Sprintf("第 %d 条「%s」：响应内容为空", i+1, kw))
			}
			continue
		}

		var existing models.KeywordResponse
		findErr := s.DB.Where("keyword = ?", kw).First(&existing).Error
		if findErr == nil {
			if mode == "skip" {
				result.Skipped++
				continue
			}
			// overwrite
			existing.Response = raw.Response
			existing.Icon = raw.Icon
			existing.Attributes = raw.Attributes
			existing.Status = status
			if err := s.DB.Save(&existing).Error; err != nil {
				result.Failed++
				if len(result.Errors) < maxErrors {
					result.Errors = append(result.Errors, fmt.Sprintf("第 %d 条「%s」：更新失败", i+1, kw))
				}
				continue
			}
			if err := s.applyImportCategories(existing.ID, raw, catByName); err != nil {
				// 分类失败不阻断主记录
				if len(result.Errors) < maxErrors {
					result.Errors = append(result.Errors, fmt.Sprintf("第 %d 条「%s」：分类关联部分失败", i+1, kw))
				}
			}
			result.Updated++
			continue
		}
		if findErr != gorm.ErrRecordNotFound {
			result.Failed++
			if len(result.Errors) < maxErrors {
				result.Errors = append(result.Errors, fmt.Sprintf("第 %d 条「%s」：查询失败", i+1, kw))
			}
			continue
		}

		m := models.KeywordResponse{
			Keyword:    kw,
			Response:   raw.Response,
			Icon:       raw.Icon,
			Attributes: raw.Attributes,
			Status:     status,
		}
		if err := s.DB.Create(&m).Error; err != nil {
			result.Failed++
			if len(result.Errors) < maxErrors {
				result.Errors = append(result.Errors, fmt.Sprintf("第 %d 条「%s」：创建失败", i+1, kw))
			}
			continue
		}
		_ = s.applyImportCategories(m.ID, raw, catByName)
		result.Created++
	}
	return result, nil
}

func (s *ContentService) applyImportCategories(keywordID uint, raw KeywordExportItem, catByName map[string]uint) error {
	var ids []uint
	if len(raw.CategoryNames) > 0 {
		for _, name := range raw.CategoryNames {
			key := strings.ToLower(strings.TrimSpace(name))
			if key == "" {
				continue
			}
			if id, ok := catByName[key]; ok {
				ids = append(ids, id)
			} else {
				// 自动创建缺失分类
				c := models.KeywordCategory{Name: strings.TrimSpace(name)}
				if err := s.DB.Create(&c).Error; err == nil {
					catByName[key] = c.ID
					ids = append(ids, c.ID)
				}
			}
		}
	} else if len(raw.CategoryIDs) > 0 {
		ids = raw.CategoryIDs
	}
	return s.KeywordCategoriesSet(keywordID, ids)
}

// ParseKeywordsCSV 解析 CSV 导入体
func ParseKeywordsCSV(r io.Reader) ([]KeywordExportItem, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true
	rows, err := cr.ReadAll()
	if err != nil {
		return nil, apperr.BadRequest("CSV 解析失败")
	}
	if len(rows) == 0 {
		return nil, apperr.BadRequest("CSV 为空")
	}
	// 支持有表头 / 无表头
	start := 0
	header := map[string]int{}
	first := rows[0]
	looksHeader := false
	for i, h := range first {
		key := strings.ToLower(strings.TrimSpace(h))
		header[key] = i
		if key == "keyword" || key == "关键词" {
			looksHeader = true
		}
	}
	if looksHeader {
		start = 1
	} else {
		// 默认列序
		header = map[string]int{
			"keyword": 0, "status": 1, "icon": 2, "attributes": 3, "categories": 4, "response": 5,
		}
	}
	col := func(row []string, name string) string {
		idx, ok := header[name]
		if !ok || idx < 0 || idx >= len(row) {
			// 中文别名
			if name == "keyword" {
				if i, ok := header["关键词"]; ok && i < len(row) {
					return strings.TrimSpace(row[i])
				}
			}
			if name == "response" {
				if i, ok := header["响应"]; ok && i < len(row) {
					return strings.TrimSpace(row[i])
				}
				if i, ok := header["内容"]; ok && i < len(row) {
					return strings.TrimSpace(row[i])
				}
			}
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	items := make([]KeywordExportItem, 0, len(rows)-start)
	for _, row := range rows[start:] {
		if len(row) == 0 {
			continue
		}
		kw := col(row, "keyword")
		if kw == "" {
			continue
		}
		catsRaw := col(row, "categories")
		if catsRaw == "" {
			catsRaw = col(row, "分类")
		}
		var catNames []string
		if catsRaw != "" {
			for _, p := range strings.Split(catsRaw, "|") {
				p = strings.TrimSpace(p)
				if p != "" {
					catNames = append(catNames, p)
				}
			}
		}
		resp := col(row, "response")
		// 若 response 列缺失且行只有 2 列，第二列当 response
		if resp == "" && len(row) >= 2 && header["response"] == 0 {
			// no-op
		}
		items = append(items, KeywordExportItem{
			Keyword:       kw,
			Status:        col(row, "status"),
			Icon:          col(row, "icon"),
			Attributes:    col(row, "attributes"),
			CategoryNames: catNames,
			Response:      resp,
		})
	}
	if len(items) == 0 {
		return nil, apperr.BadRequest("CSV 中未解析到有效关键词")
	}
	return items, nil
}
