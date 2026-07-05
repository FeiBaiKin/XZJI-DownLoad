package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"xzji/backend/internal/models"
)

type ContentService struct{ DB *gorm.DB }

// ====== 关键词 ======

type KeywordCreate struct {
	Keyword  string `json:"keyword"`
	Response string `json:"response"` // JSON 字符串或纯文本，handler 层已统一为 JSON 字符串
}

type KeywordUpdate struct {
	Keyword    *string `json:"keyword"`
	Response   *string `json:"response"`
	Icon       *string `json:"icon"`
	Attributes *string `json:"attributes"`
}

func (s *ContentService) KeywordsList() ([]models.KeywordResponse, error) {
	var list []models.KeywordResponse
	if err := s.DB.Order("id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (s *ContentService) KeywordGet(id uint) (*models.KeywordResponse, error) {
	var kw models.KeywordResponse
	if err := s.DB.First(&kw, id).Error; err != nil {
		return nil, fmt.Errorf("关键词不存在")
	}
	return &kw, nil
}

func (s *ContentService) KeywordCreate(req KeywordCreate) (*models.KeywordResponse, error) {
	kw := strings.TrimSpace(strings.ToLower(req.Keyword))
	if kw == "" {
		return nil, fmt.Errorf("关键词不能为空")
	}
	var exist int64
	s.DB.Model(&models.KeywordResponse{}).Where("keyword = ?", kw).Count(&exist)
	if exist > 0 {
		return nil, fmt.Errorf("关键词已存在")
	}
	m := models.KeywordResponse{Keyword: kw, Response: req.Response}
	if err := s.DB.Create(&m).Error; err != nil {
		return nil, fmt.Errorf("创建失败")
	}
	return &m, nil
}

func (s *ContentService) KeywordUpdate(id uint, req KeywordUpdate) (*models.KeywordResponse, error) {
	var m models.KeywordResponse
	if err := s.DB.First(&m, id).Error; err != nil {
		return nil, fmt.Errorf("关键词不存在")
	}
	if req.Keyword != nil {
		kw := strings.TrimSpace(strings.ToLower(*req.Keyword))
		if kw != m.Keyword {
			var cnt int64
			s.DB.Model(&models.KeywordResponse{}).Where("keyword = ? AND id != ?", kw, id).Count(&cnt)
			if cnt > 0 {
				return nil, fmt.Errorf("关键词已存在")
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
	if err := s.DB.Save(&m).Error; err != nil {
		return nil, fmt.Errorf("更新失败")
	}
	return &m, nil
}

func (s *ContentService) KeywordDelete(id uint) error {
	// 事务内删除关键词及其关联记录，防止孤儿数据
	return s.DB.Transaction(func(tx *gorm.DB) error {
		// 删除关键词-分类关联
		if err := tx.Where("keyword_id = ?", id).Delete(&models.KeywordCategoryLink{}).Error; err != nil {
			return fmt.Errorf("删除关联失败: %w", err)
		}
		// 删除关键词
		if err := tx.Delete(&models.KeywordResponse{}, id).Error; err != nil {
			return fmt.Errorf("删除失败: %w", err)
		}
		return nil
	})
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
		return nil, fmt.Errorf("分类名称不能为空")
	}
	m := models.KeywordCategory{Name: req.Name, Description: req.Description, SortOrder: req.SortOrder}
	if err := s.DB.Create(&m).Error; err != nil {
		return nil, fmt.Errorf("创建失败")
	}
	return &m, nil
}

func (s *ContentService) CategoryUpdate(id uint, req CategoryUpdate) (*models.KeywordCategory, error) {
	var m models.KeywordCategory
	if err := s.DB.First(&m, id).Error; err != nil {
		return nil, fmt.Errorf("分类不存在")
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
		return nil, fmt.Errorf("更新失败")
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

// PublicKeywords 公开关键词列表（支持按分类筛选 + 分页）
func (s *ContentService) PublicKeywords(categoryID *uint, page, pageSize int) (*PaginatedKeywords, error) {
	page, pageSize = normalizePage(page, pageSize, 60)
	q := s.DB.Model(&models.KeywordResponse{})
	if categoryID != nil && *categoryID > 0 {
		q = q.Where("id IN (?)",
			s.DB.Table("keyword_category_links").Select("keyword_id").Where("category_id = ?", *categoryID))
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []models.KeywordResponse
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]PublicKeyword, len(rows))
	for i, r := range rows {
		items[i] = toPublicKeyword(r)
	}
	return &PaginatedKeywords{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// PublicFeaturedKeywords 精选资源：每个分类按 id DESC 取前 perCategory 条，flatten 后整体分页。
// 精选总量有限（分类数 × perCategory），采用内存分页可接受。
func (s *ContentService) PublicFeaturedKeywords(perCategory, page, pageSize int) (*PaginatedKeywords, error) {
	if perCategory < 1 {
		perCategory = 4
	}
	page, pageSize = normalizePage(page, pageSize, 60)

	cats, err := s.CategoriesList()
	if err != nil {
		return nil, err
	}
	// 每分类取前 perCategory 条
	var all []PublicKeyword
	for _, cat := range cats {
		var rows []models.KeywordResponse
		if err := s.DB.
			Where("id IN (?)", s.DB.Table("keyword_category_links").
				Select("keyword_id").Where("category_id = ?", cat.ID)).
			Order("id DESC").Limit(perCategory).Find(&rows).Error; err != nil {
			return nil, err
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
		return nil, fmt.Errorf("页面不存在")
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
	"hello":  `"你好！欢迎使用关键词搜索应用。"`,
	"node":   `"Node.js 是一个基于 Chrome V8 引擎的 JavaScript 运行时。"`,
	"react":  `"React 是一个用于构建用户界面的 JavaScript 库。"`,
	"mysql":  `"MySQL 是最流行的关系型数据库管理系统之一。"`,
	"search": `"搜索功能已启用，正在探索关键词的奥秘。"`,
}

func (s *ContentService) SearchKeyword(keyword string) (*SearchResult, error) {
	kw := strings.TrimSpace(strings.ToLower(keyword))
	var m models.KeywordResponse
	err := s.DB.Where("keyword = ?", kw).First(&m).Error
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
	if e := db.First(&m, keywordID).Error; e != nil {
		return nil, fmt.Errorf("关键词不存在")
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
