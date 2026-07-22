package migrate

import (
	"music-discovery/backend/entity"

	"gorm.io/gorm"
)

func CreateRefreshTokens(db *gorm.DB) error {
	if !db.Migrator().HasTable(&entity.RefreshToken{}) {
		if err := db.AutoMigrate(&entity.RefreshToken{}); err != nil {
			return err
		}
	}

	// Both fresh and existing databases pass through the same normalization.
	// Existing databases may contain rows created before token_version and
	// revoked_at rotation were introduced. Prepare and backfill first so
	// AutoMigrate never attempts to add a NOT NULL column to populated data.
	statements := []string{
		`ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS token_version BIGINT`,
		`ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMPTZ NULL`,
		`UPDATE refresh_tokens rt SET token_version = u.token_version FROM users u WHERE rt.user_id = u.id AND rt.token_version IS NULL`,
		`UPDATE refresh_tokens SET token_version = 0 WHERE token_version IS NULL`,
		`UPDATE refresh_tokens SET revoked_at = used_at WHERE revoked_at IS NULL AND used_at IS NOT NULL`,
		`ALTER TABLE refresh_tokens ALTER COLUMN token_version SET DEFAULT 0`,
		`ALTER TABLE refresh_tokens ALTER COLUMN token_version SET NOT NULL`,
		`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_refresh_tokens_token_version') THEN ALTER TABLE refresh_tokens ADD CONSTRAINT chk_refresh_tokens_token_version CHECK (token_version >= 0); END IF; END $$`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return db.AutoMigrate(&entity.RefreshToken{})
}
