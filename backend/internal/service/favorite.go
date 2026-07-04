package service

import (
	"time"

	"gorm.io/gorm"
	"xzji/backend/internal/models"
)

// FavoriteService 关键词收藏服务（依赖注入 DB）
type FavoriteService struct{ DB *gorm.DB }

// ToggleFavorite 切换收藏状态（已收藏则取消，未收藏则新增）。返回当前是否已收藏。
func (s *FavoriteService) ToggleFavorite(userID, keywordID uint) (bool, error) {
	var existing models.KeywordFavorite
	err := s.DB.Where("user_id = ? AND keyword_id = ?", userID, keywordID).First(&existing).Error
	if err == nil {
		// 已存在 -> 取消收藏
		if delErr := s.DB.Delete(&existing).Error; delErr != nil {
			return false, delErr
		}
		return false, nil
	}
	if err != gorm.ErrRecordNotFound {
		return false, err
	}
	// 未收藏 -> 新增（忽略并发冲突下的唯一索引报错）
	if createErr := s.DB.Create(&models.KeywordFavorite{
		UserID:    userID,
		KeywordID: keywordID,
		CreatedAt: time.Now(),
	}).Error; createErr != nil {
		return false, createErr
	}
	return true, nil
}

// ListFavorites 分页返回用户收藏的关键词（按收藏时间倒序，join keyword_responses）。
// 返回标准 PaginatedKeywords 结构，供前端复用 PublicKeyword 渲染。
func (s *FavoriteService) ListFavorites(userID uint, page, pageSize int) (PaginatedKeywords, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 60 {
		pageSize = 8
	}

	var total int64
	if err := s.DB.Model(&models.KeywordFavorite{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return PaginatedKeywords{}, err
	}

	// 先取当前页的 keyword_id + favorited_at，再 join keyword_responses 取预览
	type favRow struct {
		KeywordID   uint      `gorm:"column:keyword_id"`
		FavoritedAt time.Time `gorm:"column:favorited_at"`
	}
	var rows []favRow
	offset := (page - 1) * pageSize
	if err := s.DB.Table("keyword_favorites").
		Select("keyword_id, created_at AS favorited_at").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&rows).Error; err != nil {
		return PaginatedKeywords{}, err
	}

	ids := make([]uint, 0, len(rows))
	favTimeMap := make(map[uint]time.Time, len(rows))
	for _, r := range rows {
		ids = append(ids, r.KeywordID)
		favTimeMap[r.KeywordID] = r.FavoritedAt
	}

	items := make([]PublicKeyword, 0, len(rows))
	if len(ids) > 0 {
		var kws []models.KeywordResponse
		if err := s.DB.Where("id IN ?", ids).Find(&kws).Error; err != nil {
			return PaginatedKeywords{}, err
		}
		// 按 favRow 原始顺序（收藏时间倒序）输出
		kwMap := make(map[uint]models.KeywordResponse, len(kws))
		for _, k := range kws {
			kwMap[k.ID] = k
		}
		for _, id := range ids {
			if k, ok := kwMap[id]; ok {
				pk := toPublicKeyword(k)
				// 借用 CreatedAt 字段透传收藏时间，便于前端展示"收藏于 X"
				pk.CreatedAt = favTimeMap[id]
				items = append(items, pk)
			}
		}
	}

	return PaginatedKeywords{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// ListFavoriteIDs 返回用户全部收藏的 keyword_id 列表（供前端 hydrate 收藏状态）。
func (s *FavoriteService) ListFavoriteIDs(userID uint) ([]uint, error) {
	var ids []uint
	if err := s.DB.Model(&models.KeywordFavorite{}).
		Where("user_id = ?", userID).
		Pluck("keyword_id", &ids).Error; err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []uint{}
	}
	return ids, nil
}
