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
	if err := SeedTags(db); err != nil {
		return fmt.Errorf("seed tags: %w", err)
	}
	return nil
}
