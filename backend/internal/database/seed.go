package database

import (
	"log/slog"

	"xzji/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SeedInstallData 仅在安装向导完成建表后写入最小演示数据。
// 正常启动不得调用此函数，避免系统重启后补写任何用户已删除的数据。
func SeedInstallData(db *gorm.DB) error {
	hello := models.KeywordResponse{
		Keyword:  "hello",
		Response: "你好！欢迎使用关键词搜索应用。",
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&hello).Error; err != nil {
		return err
	}
	slog.Info("已插入最小安装数据", "keyword", hello.Keyword)
	return nil
}
