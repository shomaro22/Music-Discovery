package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestPostgreSQLUserConstraints(t *testing.T) {
	db := openTestPostgres(t)
	now := time.Now().UTC()
	insert := func(id uuid.UUID, name, email, hash, role, status string, tokenVersion int64) error {
		return db.Exec(`INSERT INTO users (id,name,email,password_hash,role,status,token_version,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)`, id, name, email, hash, role, status, tokenVersion, now, now).Error
	}
	if err := insert(uuid.New(), "Taro", "base@example.com", "hash", "user", "active", 0); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		run  func(*gorm.DB) error
	}{
		{"duplicate-email", func(db *gorm.DB) error {
			return insert(uuid.New(), "Jiro", "base@example.com", "hash", "user", "active", 0)
		}},
		{"invalid-role", func(db *gorm.DB) error {
			return insert(uuid.New(), "Jiro", "role@example.com", "hash", "admin", "active", 0)
		}},
		{"invalid-status", func(db *gorm.DB) error {
			return insert(uuid.New(), "Jiro", "status@example.com", "hash", "user", "deleted", 0)
		}},
		{"negative-token-version", func(db *gorm.DB) error {
			return insert(uuid.New(), "Jiro", "token@example.com", "hash", "user", "active", -1)
		}},
		{"empty-name", func(db *gorm.DB) error {
			return insert(uuid.New(), "", "name@example.com", "hash", "user", "active", 0)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(db); err == nil {
				t.Fatal("expected constraint violation")
			}
		})
	}
}
