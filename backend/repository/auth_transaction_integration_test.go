package repository

import (
	"context"
	"testing"
	"time"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
)

func TestAuthTransactionManagerRollbackKeepsFamilyActive(t *testing.T) {
	db := openTestPostgres(t)
	if err := db.AutoMigrate(&entity.RefreshToken{}); err != nil {
		t.Fatalf("migrate refresh_tokens: %v", err)
	}
	if err := db.Exec("TRUNCATE TABLE refresh_tokens, users RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}

	user := validUser("rollback@example.com")
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	familyID := uuid.New()
	token := &entity.RefreshToken{ID: uuid.New(), UserID: user.ID, TokenHash: "rollback-hash", FamilyID: familyID, ExpiresAt: time.Now().UTC().Add(time.Hour), CreatedAt: time.Now().UTC()}
	if err := db.Create(token).Error; err != nil {
		t.Fatalf("create token: %v", err)
	}

	err := NewAuthTransactionManager(db).RevokeFamilyAndIncrementTokenVersion(context.Background(), familyID, uuid.New(), time.Now().UTC())
	if err == nil {
		t.Fatal("expected transaction error")
	}

	var stored entity.RefreshToken
	if err := db.First(&stored, "id = ?", token.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.RevokedAt != nil {
		t.Fatal("refresh family update was not rolled back")
	}
	var storedUser entity.User
	if err := db.First(&storedUser, "id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedUser.TokenVersion != 0 {
		t.Fatalf("token_version=%d", storedUser.TokenVersion)
	}
}

func TestAuthTransactionManagerUpdatesFamilyAndVersionAtomically(t *testing.T) {
	db := openTestPostgres(t)
	if err := db.AutoMigrate(&entity.RefreshToken{}); err != nil {
		t.Fatalf("migrate refresh_tokens: %v", err)
	}
	if err := db.Exec("TRUNCATE TABLE refresh_tokens, users RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}

	user := validUser("atomic@example.com")
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	familyID := uuid.New()
	token := &entity.RefreshToken{ID: uuid.New(), UserID: user.ID, TokenHash: "atomic-hash", FamilyID: familyID, ExpiresAt: time.Now().UTC().Add(time.Hour), CreatedAt: time.Now().UTC()}
	if err := db.Create(token).Error; err != nil {
		t.Fatalf("create token: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	if err := NewAuthTransactionManager(db).RevokeFamilyAndIncrementTokenVersion(context.Background(), familyID, user.ID, now); err != nil {
		t.Fatal(err)
	}
	var stored entity.RefreshToken
	if err := db.First(&stored, "id = ?", token.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.RevokedAt == nil {
		t.Fatal("family was not revoked")
	}
	var storedUser entity.User
	if err := db.First(&storedUser, "id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedUser.TokenVersion != 1 {
		t.Fatalf("token_version=%d", storedUser.TokenVersion)
	}
}

func TestAuthTransactionManagerDoesNotIncrementVersionWhenFamilyAlreadyRevoked(t *testing.T) {
	db := openTestPostgres(t)
	if err := db.AutoMigrate(&entity.RefreshToken{}); err != nil {
		t.Fatalf("migrate refresh_tokens: %v", err)
	}
	if err := db.Exec("TRUNCATE TABLE refresh_tokens, users RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}

	user := validUser("already-revoked@example.com")
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	familyID := uuid.New()
	token := &entity.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: "already-revoked-hash",
		FamilyID:  familyID,
		UsedAt:    &now,
		RevokedAt: &now,
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now.Add(-time.Minute),
	}
	if err := db.Create(token).Error; err != nil {
		t.Fatalf("create token: %v", err)
	}

	err := NewAuthTransactionManager(db).RevokeFamilyAndIncrementTokenVersion(context.Background(), familyID, user.ID, now.Add(time.Second))
	if err != entity.ErrInvalidRefreshToken {
		t.Fatalf("err=%v want=%v", err, entity.ErrInvalidRefreshToken)
	}

	var storedUser entity.User
	if err := db.First(&storedUser, "id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedUser.TokenVersion != 0 {
		t.Fatalf("token_version=%d want=0", storedUser.TokenVersion)
	}
}

func TestAuthTransactionManagerDoesNotRevokeDifferentFamily(t *testing.T) {
	db := openTestPostgres(t)
	if err := db.AutoMigrate(&entity.RefreshToken{}); err != nil {
		t.Fatalf("migrate refresh_tokens: %v", err)
	}
	if err := db.Exec("TRUNCATE TABLE refresh_tokens, users RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	user := validUser("different-family@example.com")
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	targetFamily := uuid.New()
	otherFamily := uuid.New()
	target := &entity.RefreshToken{ID: uuid.New(), UserID: user.ID, TokenHash: "target-family", FamilyID: targetFamily, ExpiresAt: now.Add(time.Hour), CreatedAt: now}
	other := &entity.RefreshToken{ID: uuid.New(), UserID: user.ID, TokenHash: "other-family", FamilyID: otherFamily, ExpiresAt: now.Add(time.Hour), CreatedAt: now}
	if err := db.Create([]*entity.RefreshToken{target, other}).Error; err != nil {
		t.Fatal(err)
	}
	if err := NewAuthTransactionManager(db).RevokeFamilyAndIncrementTokenVersion(context.Background(), targetFamily, user.ID, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	var storedOther entity.RefreshToken
	if err := db.First(&storedOther, "id = ?", other.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedOther.RevokedAt != nil {
		t.Fatalf("different family was revoked: %+v", storedOther)
	}
}
