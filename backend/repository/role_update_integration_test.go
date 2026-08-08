package repository

import (
	"context"
	"testing"
	"time"

	"music-discovery/backend/entity"
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
	manager := NewAuthTransactionManager(db)
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

	if err := db.Exec(`
		CREATE OR REPLACE FUNCTION fail_role_update() RETURNS trigger AS $$
		BEGIN
			RAISE EXCEPTION 'forced role update failure';
		END;
		$$ LANGUAGE plpgsql;
	`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
		CREATE TRIGGER users_fail_role_update
		AFTER UPDATE OF role ON users
		FOR EACH ROW EXECUTE FUNCTION fail_role_update();
	`).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Exec("DROP TRIGGER IF EXISTS users_fail_role_update ON users").Error
		_ = db.Exec("DROP FUNCTION IF EXISTS fail_role_update()").Error
	})

	manager := NewAuthTransactionManager(db)
	err := manager.ChangeUserRole(context.Background(), user.ID, entity.RoleContentAdmin, time.Now().UTC())
	if err == nil {
		t.Fatal("expected role update failure")
	}

	var stored entity.User
	if err := db.First(&stored, "id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Role != user.Role || stored.TokenVersion != user.TokenVersion || !stored.UpdatedAt.Equal(originalUpdatedAt) {
		t.Fatalf("stored=%+v", stored)
	}
}
