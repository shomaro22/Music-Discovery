package repository

import (
	"context"
	"testing"
	"time"

	"music-discovery/backend/entity"

	"gorm.io/gorm"
)

func TestChangeUserRoleUpdatesRoleVersionAndTimestamp(t *testing.T) {
	db := openTestPostgres(t)
	if err := db.Exec("TRUNCATE TABLE users CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	user := validUser("role-update@example.com")
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 23, 4, 0, 0, 0, time.UTC)
	manager := NewAuthTransactionManager(db).(IRoleTransactionManager)
	if err := manager.ChangeUserRole(context.Background(), user.ID, entity.RoleContentAdmin, now); err != nil {
		t.Fatal(err)
	}
	var stored entity.User
	if err := db.First(&stored, "id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Role != entity.RoleContentAdmin || stored.TokenVersion != user.TokenVersion+1 || !stored.UpdatedAt.Equal(now) {
		t.Fatalf("stored=%+v", stored)
	}
}

func TestChangeUserRoleRollsBackOnTransactionFailure(t *testing.T) {
	db := openTestPostgres(t)
	if err := db.Exec("TRUNCATE TABLE users CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	user := validUser("role-rollback@example.com")
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	originalUpdatedAt := user.UpdatedAt
	err := db.Transaction(func(tx *gorm.DB) error {
		result := tx.Exec("UPDATE users SET role = ?, token_version = token_version + 1, updated_at = ? WHERE id = ?", entity.RoleContentAdmin, time.Now().UTC(), user.ID)
		if result.Error != nil {
			return result.Error
		}
		return entity.ErrInternal
	})
	if err == nil {
		t.Fatal("expected rollback error")
	}
	var stored entity.User
	if err := db.First(&stored, "id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Role != user.Role || stored.TokenVersion != user.TokenVersion || !stored.UpdatedAt.Equal(originalUpdatedAt) {
		t.Fatalf("stored=%+v", stored)
	}
}
