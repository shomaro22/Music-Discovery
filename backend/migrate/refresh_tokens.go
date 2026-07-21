package migrate

import (
	"music-discovery/backend/entity"

	"gorm.io/gorm"
)

func CreateRefreshTokens(db *gorm.DB) error {
	if err := db.AutoMigrate(&entity.RefreshToken{}); err != nil {
		return err
	}
	return db.Exec(`ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS token_version BIGINT NOT NULL DEFAULT 0 CHECK (token_version >= 0)`).Error
}
