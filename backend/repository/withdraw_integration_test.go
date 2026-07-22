package repository

import (
	"context"
	"testing"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/migrate"

	"github.com/google/uuid"
)

func TestWithdrawTransactionUsesOneTimestampAndDoesNotAffectOtherUser(t *testing.T) {
	db := openTestPostgres(t)
	if err := migrate.CreateRefreshTokens(db); err != nil {
		t.Fatalf("migrate refresh tokens: %v", err)
	}
	if err := db.Exec("TRUNCATE TABLE refresh_tokens, users").Error; err != nil {
		t.Fatal(err)
	}

	target := validUser("withdraw-target@example.com")
	other := validUser("withdraw-other@example.com")
	if err := db.Create(target).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(other).Error; err != nil {
		t.Fatal(err)
	}
	created := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	tokens := []entity.RefreshToken{
		{ID: uuid.New(), UserID: target.ID, TokenHash: uuid.NewString(), FamilyID: uuid.New(), ExpiresAt: created.Add(24 * time.Hour), CreatedAt: created, UpdatedAt: created},
		{ID: uuid.New(), UserID: target.ID, TokenHash: uuid.NewString(), FamilyID: uuid.New(), ExpiresAt: created.Add(24 * time.Hour), CreatedAt: created, UpdatedAt: created},
		{ID: uuid.New(), UserID: other.ID, TokenHash: uuid.NewString(), FamilyID: uuid.New(), ExpiresAt: created.Add(24 * time.Hour), CreatedAt: created, UpdatedAt: created},
	}
	if err := db.Create(&tokens).Error; err != nil {
		t.Fatal(err)
	}

	userRepo := NewUserRepository(db)
	refreshRepo := NewRefreshTokenRepository(db)
	txManager := NewAuthTransactionManager(db).(ITransactionManager)
	now := time.Date(2026, 7, 22, 12, 34, 56, 123000000, time.UTC)
	err := txManager.WithinTransaction(context.Background(), func(tx ITransaction) error {
		if err := userRepo.SoftDelete(context.Background(), tx, target.ID, now); err != nil {
			return err
		}
		if err := userRepo.IncrementTokenVersion(context.Background(), tx, target.ID, now); err != nil {
			return err
		}
		return refreshRepo.RevokeAllByUserID(context.Background(), tx, target.ID, now)
	})
	if err != nil {
		t.Fatal(err)
	}

	var stored entity.User
	if err := db.Unscoped().First(&stored, "id = ?", target.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !stored.DeletedAt.Time.Equal(now) || !stored.UpdatedAt.Equal(now) || stored.TokenVersion != 1 {
		t.Fatalf("target timestamps/version: deleted=%v updated=%v version=%d", stored.DeletedAt.Time, stored.UpdatedAt, stored.TokenVersion)
	}
	var targetTokens []entity.RefreshToken
	if err := db.Find(&targetTokens, "user_id = ?", target.ID).Error; err != nil {
		t.Fatal(err)
	}
	for _, token := range targetTokens {
		if token.RevokedAt == nil || !token.RevokedAt.Equal(now) || !token.UpdatedAt.Equal(now) {
			t.Fatalf("target token timestamps: %+v", token)
		}
	}
	var otherToken entity.RefreshToken
	if err := db.First(&otherToken, "user_id = ?", other.ID).Error; err != nil {
		t.Fatal(err)
	}
	if otherToken.RevokedAt != nil || !otherToken.UpdatedAt.Equal(created) {
		t.Fatalf("other token changed: %+v", otherToken)
	}
}

func TestWithdrawTransactionRollsBackWhenLaterStepFails(t *testing.T) {
	db := openTestPostgres(t)
	if err := migrate.CreateRefreshTokens(db); err != nil {
		t.Fatalf("migrate refresh tokens: %v", err)
	}
	if err := db.Exec("TRUNCATE TABLE refresh_tokens, users").Error; err != nil {
		t.Fatal(err)
	}

	user := validUser("withdraw-rollback-after-delete@example.com")
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	originalUpdatedAt := user.UpdatedAt
	now := time.Date(2026, 7, 23, 1, 2, 3, 0, time.UTC)
	userRepo := NewUserRepository(db)
	txManager := NewAuthTransactionManager(db)

	err := txManager.WithinTransaction(context.Background(), func(tx ITransaction) error {
		if err := userRepo.SoftDelete(context.Background(), tx, user.ID, now); err != nil {
			return err
		}
		return entity.ErrInternal
	})
	if err == nil {
		t.Fatal("expected transaction error")
	}

	var stored entity.User
	if err := db.Unscoped().First(&stored, "id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.DeletedAt.Valid {
		t.Fatalf("deleted_at must be rolled back: %v", stored.DeletedAt.Time)
	}
	if stored.TokenVersion != user.TokenVersion {
		t.Fatalf("token_version changed: got %d want %d", stored.TokenVersion, user.TokenVersion)
	}
	if !stored.UpdatedAt.Equal(originalUpdatedAt) {
		t.Fatalf("updated_at changed: got %v want %v", stored.UpdatedAt, originalUpdatedAt)
	}
}

func TestWithdrawTransactionRollsBackUserAndRefreshTokenUpdates(t *testing.T) {
	db := openTestPostgres(t)
	if err := migrate.CreateRefreshTokens(db); err != nil {
		t.Fatalf("migrate refresh tokens: %v", err)
	}
	if err := db.Exec("TRUNCATE TABLE refresh_tokens, users").Error; err != nil {
		t.Fatal(err)
	}

	user := validUser("withdraw-rollback-all@example.com")
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	created := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	token := entity.RefreshToken{
		ID: uuid.New(), UserID: user.ID, TokenHash: uuid.NewString(), FamilyID: uuid.New(),
		ExpiresAt: created.Add(24 * time.Hour), CreatedAt: created, UpdatedAt: created,
	}
	if err := db.Create(&token).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 7, 23, 2, 3, 4, 0, time.UTC)
	userRepo := NewUserRepository(db)
	refreshRepo := NewRefreshTokenRepository(db)
	txManager := NewAuthTransactionManager(db)

	err := txManager.WithinTransaction(context.Background(), func(tx ITransaction) error {
		if err := userRepo.SoftDelete(context.Background(), tx, user.ID, now); err != nil {
			return err
		}
		if err := userRepo.IncrementTokenVersion(context.Background(), tx, user.ID, now); err != nil {
			return err
		}
		if err := refreshRepo.RevokeAllByUserID(context.Background(), tx, user.ID, now); err != nil {
			return err
		}
		return entity.ErrInternal
	})
	if err == nil {
		t.Fatal("expected transaction error")
	}

	var storedUser entity.User
	if err := db.Unscoped().First(&storedUser, "id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedUser.DeletedAt.Valid {
		t.Fatalf("deleted_at must be rolled back: %v", storedUser.DeletedAt.Time)
	}
	if storedUser.TokenVersion != user.TokenVersion {
		t.Fatalf("token_version changed: got %d want %d", storedUser.TokenVersion, user.TokenVersion)
	}

	var storedToken entity.RefreshToken
	if err := db.First(&storedToken, "id = ?", token.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedToken.RevokedAt != nil {
		t.Fatalf("revoked_at must be rolled back: %v", storedToken.RevokedAt)
	}
	if !storedToken.UpdatedAt.Equal(created) {
		t.Fatalf("refresh token updated_at changed: got %v want %v", storedToken.UpdatedAt, created)
	}
}
