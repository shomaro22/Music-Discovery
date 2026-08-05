package migrate

import (
	"fmt"
	"gorm.io/gorm"
	"music-discovery/backend/entity"
)

func CreateAdminTables(db *gorm.DB) error {
	if err := db.AutoMigrate(&entity.Artist{}, &entity.Song{}, &entity.Tag{}, &entity.ArtistTag{}, &entity.SongTag{}, &entity.AuditLog{}, &entity.SystemLog{}); err != nil {
		return fmt.Errorf("migrate admin tables: %w", err)
	}
	// review_status導入前から存在するSongは、管理者確認済みの既存
	// コンテンツとして扱う。
	if err := db.Model(&entity.Song{}).
		Where("review_status IS NULL OR review_status = ''").
		Update("review_status", entity.ContentReviewStatusApproved).Error; err != nil {
		return fmt.Errorf("backfill song review status: %w", err)
	}
	if err := SeedTags(db); err != nil {
		return fmt.Errorf("seed tags: %w", err)
	}
	return nil
}
