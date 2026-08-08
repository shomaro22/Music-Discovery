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

	// 新規DBと既存DBの両方に対して、同じ正規化処理を実行する。
	// 既存DBには、token_versionやrevoked_atによるローテーションを導入する前に作成されたレコードが存在する可能性がある。
	// データが入ったテーブルへAutoMigrateが直接NOT NULLカラムを追加しないように、先にカラムの準備と既存データの補完を行う。
	statements := []string{
		`ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS token_version BIGINT`,
		`ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMPTZ NULL`,
		`ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ`,
		`UPDATE refresh_tokens rt SET token_version = u.token_version FROM users u WHERE rt.user_id = u.id AND rt.token_version IS NULL`,
		`UPDATE refresh_tokens SET token_version = 0 WHERE token_version IS NULL`,
		`UPDATE refresh_tokens SET revoked_at = used_at WHERE revoked_at IS NULL AND used_at IS NOT NULL`,
		`UPDATE refresh_tokens SET updated_at = created_at WHERE updated_at IS NULL`,
		`ALTER TABLE refresh_tokens ALTER COLUMN updated_at SET NOT NULL`,
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
