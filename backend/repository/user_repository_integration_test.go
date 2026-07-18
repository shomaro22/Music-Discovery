package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/migrate"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func openTestPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; PostgreSQL integration test skipped")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	if err := migrate.CreateUsers(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("TRUNCATE TABLE users").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return db
}

func validUser(email string) *entity.User {
	now := time.Now().UTC()
	return &entity.User{ID: uuid.New(), Name: "Taro", Email: email, PasswordHash: "hashed", Role: entity.UserRoleUser, Status: entity.UserStatusActive, TokenVersion: 0, CreatedAt: now, UpdatedAt: now}
}

func TestUserRepositoryCreateFindAndDuplicate(t *testing.T) {
	repo := NewUserRepository(openTestPostgres(t))
	ctx := context.Background()
	created, err := repo.Create(ctx, validUser("taro@example.com"))
	if err != nil || created.ID == uuid.Nil {
		t.Fatalf("create: user=%#v err=%v", created, err)
	}
	found, err := repo.FindByEmail(ctx, "taro@example.com")
	if err != nil || found.ID != created.ID {
		t.Fatalf("find: user=%#v err=%v", found, err)
	}
	_, err = repo.Create(ctx, validUser("taro@example.com"))
	if !errors.Is(err, entity.ErrEmailAlreadyExists) {
		t.Fatalf("duplicate err=%v", err)
	}
}

func TestUserRepositoryValidation(t *testing.T) {
	repo := NewUserRepository(openTestPostgres(t))
	tests := []*entity.User{nil, validUser(""), validUser("a@example.com"), validUser("b@example.com")}
	tests[2].PasswordHash = ""
	tests[3].Role = ""
	for i, user := range tests {
		_, err := repo.Create(context.Background(), user)
		if !errors.Is(err, entity.ErrInvalidUserForCreate) {
			t.Fatalf("case %d err=%v", i, err)
		}
	}
}

func TestUserRepositoryHonorsCanceledContext(t *testing.T) {
	repo := NewUserRepository(openTestPostgres(t))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := repo.FindByEmail(ctx, "nobody@example.com")
	if err == nil {
		t.Fatal("expected context-related repository error")
	}
}
