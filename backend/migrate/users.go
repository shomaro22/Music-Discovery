package migrate

import (
	"fmt"

	"gorm.io/gorm"
)

func CreateUsers(db *gorm.DB) error {
	statements := []string{
		`CREATE EXTENSION IF NOT EXISTS "pgcrypto"`,
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(50) NOT NULL CHECK (char_length(name) BETWEEN 1 AND 50),
			email VARCHAR(254) NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role VARCHAR(32) NOT NULL CHECK (role IN ('user', 'content_admin', 'system_admin')),
			status VARCHAR(32) NOT NULL CHECK (status IN ('active', 'suspended')),
			token_version BIGINT NOT NULL DEFAULT 0 CHECK (token_version >= 0),
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			deleted_at TIMESTAMPTZ NULL
		)`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}
