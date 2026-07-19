package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/migrate"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestRefreshTokenRepositoryCreateIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := migrate.CreateUsers(database); err != nil {
		t.Fatal(err)
	}
	if err := migrate.CreateRefreshTokens(database); err != nil {
		t.Fatal(err)
	}
	_ = database.Exec("TRUNCATE TABLE refresh_tokens, users CASCADE").Error
	user := validUser("refresh@example.com")
	created, err := NewUserRepository(database).Create(context.Background(), user)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	token := &entity.RefreshToken{UserID: created.ID, TokenHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", FamilyID: uuid.New(), ExpiresAt: now.Add(time.Hour), CreatedAt: now}
	if err := NewRefreshTokenRepository(database).Create(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	var stored entity.RefreshToken
	if err := database.First(&stored, "token_hash = ?", token.TokenHash).Error; err != nil {
		t.Fatal(err)
	}
	if stored.ID == uuid.Nil || stored.UserID != created.ID || stored.FamilyID != token.FamilyID {
		t.Fatalf("stored=%#v", stored)
	}
}
