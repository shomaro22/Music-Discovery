package migrate

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const developmentSeedPassword = "Password123!"

func SeedDevelopmentUsers(db *gorm.DB) error {
	return SeedDevelopmentUsersForEnvironment(db, os.Getenv("APP_ENV"))
}

func SeedDevelopmentUsersForEnvironment(db *gorm.DB, appEnv string) error {
	if strings.EqualFold(strings.TrimSpace(appEnv), "production") {
		return errors.New("development seed users are disabled in production")
	}
	if db == nil {
		return errors.New("database is required")
	}
	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(developmentSeedPassword),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return fmt.Errorf("hash development seed password: %w", err)
	}

	now := time.Now().UTC()
	users := []entity.User{
		{
			ID:           uuid.New(),
			Name:         "Test User",
			Email:        "user@example.com",
			PasswordHash: string(passwordHash),
			Role:         entity.UserRoleUser,
			Status:       entity.UserStatusActive,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           uuid.New(),
			Name:         "Content Admin",
			Email:        "content-admin@example.com",
			PasswordHash: string(passwordHash),
			Role:         entity.UserRoleContentAdmin,
			Status:       entity.UserStatusActive,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           uuid.New(),
			Name:         "System Admin",
			Email:        "system-admin@example.com",
			PasswordHash: string(passwordHash),
			Role:         entity.UserRoleSystemAdmin,
			Status:       entity.UserStatusActive,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for i := range users {
			users[i].Email = strings.ToLower(strings.TrimSpace(users[i].Email))
			result := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "email"}},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"name":          users[i].Name,
					"password_hash": users[i].PasswordHash,
					"role":          users[i].Role,
					"status":        users[i].Status,
					"token_version": gorm.Expr("users.token_version + 1"),
					"updated_at":    now,
				}),
			}).Create(&users[i])
			if result.Error != nil {
				return fmt.Errorf("seed development user %s: %w", users[i].Email, result.Error)
			}
		}
		return nil
	})
}
