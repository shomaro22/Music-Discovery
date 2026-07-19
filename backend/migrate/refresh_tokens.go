package migrate

import (
	"gorm.io/gorm"
	"music-discovery/backend/entity"
)

func CreateRefreshTokens(db *gorm.DB) error { return db.AutoMigrate(&entity.RefreshToken{}) }
