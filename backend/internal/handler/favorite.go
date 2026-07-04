package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"xzji/backend/internal/middleware"
	"xzji/backend/internal/service"
)

// favoriteSvc 从 context 获取 FavoriteService（依赖注入）
func favoriteSvc(c *gin.Context) *service.FavoriteService {
	return getApp(c).FavoriteSvc
}

// ToggleFavorite 切换收藏：POST /user/favorites/:keywordId → { favorited: bool }
func ToggleFavorite(c *gin.Context) {
	userID := middleware.GetUserID(c)
	idStr := c.Param("keywordId")
	keywordID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || keywordID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的关键词 ID"})
		return
	}
	fav, err := favoriteSvc(c).ToggleFavorite(userID, uint(keywordID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"favorited": fav})
}

// ListFavorites 收藏列表：GET /user/favorites?page=1&pageSize=8
func ListFavorites(c *gin.Context) {
	userID := middleware.GetUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "8"))
	result, err := favoriteSvc(c).ListFavorites(userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取收藏列表失败"})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ListFavoriteIDs 收藏 ID 列表：GET /user/favorites/ids → [1,2,3]
func ListFavoriteIDs(c *gin.Context) {
	userID := middleware.GetUserID(c)
	ids, err := favoriteSvc(c).ListFavoriteIDs(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取收藏 ID 失败"})
		return
	}
	c.JSON(http.StatusOK, ids)
}
