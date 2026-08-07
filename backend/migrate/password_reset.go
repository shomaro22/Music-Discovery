package migrate

import (
	"music-discovery/backend/entity"

	"gorm.io/gorm"
)

func CreatePasswordResetTokens(db *gorm.DB) error {
	return db.AutoMigrate(&entity.PasswordResetToken{})
}
