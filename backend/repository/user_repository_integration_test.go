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
	now := time.Now().UTC().Truncate(time.Microsecond)
	return &entity.User{
		Name:         "Taro",
		Email:        email,
		PasswordHash: "hashed-password",
		Role:         entity.UserRoleUser,
		Status:       entity.UserStatusActive,
		TokenVersion: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestUserRepositoryCreateFindAndDuplicate(t *testing.T) {
	repo := NewUserRepository(openTestPostgres(t))
	ctx := context.Background()
	input := validUser("taro@example.com")

	created, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created == nil {
		t.Fatal("create returned nil user")
	}
	if created.ID == uuid.Nil {
		t.Fatal("created user ID is nil")
	}
	if created.Name != input.Name {
		t.Fatalf("created name=%q want=%q", created.Name, input.Name)
	}
	if created.Email != input.Email {
		t.Fatalf("created email=%q want=%q", created.Email, input.Email)
	}
	if created.PasswordHash != input.PasswordHash {
		t.Fatalf("created password_hash=%q want=%q", created.PasswordHash, input.PasswordHash)
	}
	if created.Role != entity.UserRoleUser {
		t.Fatalf("created role=%q want=%q", created.Role, entity.UserRoleUser)
	}
	if created.Status != entity.UserStatusActive {
		t.Fatalf("created status=%q want=%q", created.Status, entity.UserStatusActive)
	}

	found, err := repo.FindByEmail(ctx, input.Email)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if found == nil {
		t.Fatal("find returned nil user")
	}
	if found.ID != created.ID {
		t.Fatalf("found id=%s want=%s", found.ID, created.ID)
	}
	if found.Name != input.Name {
		t.Fatalf("found name=%q want=%q", found.Name, input.Name)
	}
	if found.Email != input.Email {
		t.Fatalf("found email=%q want=%q", found.Email, input.Email)
	}
	if found.PasswordHash != input.PasswordHash {
		t.Fatalf("found password_hash=%q want=%q", found.PasswordHash, input.PasswordHash)
	}
	if found.Role != entity.UserRoleUser {
		t.Fatalf("found role=%q want=%q", found.Role, entity.UserRoleUser)
	}
	if found.Status != entity.UserStatusActive {
		t.Fatalf("found status=%q want=%q", found.Status, entity.UserStatusActive)
	}

	_, err = repo.Create(ctx, validUser(input.Email))
	if !errors.Is(err, entity.ErrEmailAlreadyExists) {
		t.Fatalf("duplicate err=%v", err)
	}
}

func TestUserRepositoryFindByEmailNotFound(t *testing.T) {
	repo := NewUserRepository(openTestPostgres(t))

	user, err := repo.FindByEmail(context.Background(), "missing@example.com")
	if user != nil {
		t.Fatalf("user=%#v want=nil", user)
	}
	if !errors.Is(err, entity.ErrUserNotFound) {
		t.Fatalf("err=%v want=%v", err, entity.ErrUserNotFound)
	}
}

func TestUserRepositoryValidation(t *testing.T) {
	repo := NewUserRepository(openTestPostgres(t))

	tests := []struct {
		name string
		user *entity.User
	}{
		{name: "nil user", user: nil},
		{name: "empty email", user: validUser("")},
		{name: "empty password hash", user: func() *entity.User {
			user := validUser("password@example.com")
			user.PasswordHash = ""
			return user
		}()},
		{name: "empty role", user: func() *entity.User {
			user := validUser("role@example.com")
			user.Role = ""
			return user
		}()},
		{name: "empty status", user: func() *entity.User {
			user := validUser("status@example.com")
			user.Status = ""
			return user
		}()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := repo.Create(context.Background(), tt.user)
			if !errors.Is(err, entity.ErrInvalidUserForCreate) {
				t.Fatalf("err=%v want=%v", err, entity.ErrInvalidUserForCreate)
			}
		})
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
