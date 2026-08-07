package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/repository"

	"github.com/google/uuid"
)

type passwordResetUserRepositoryStub struct {
	user *entity.User
	err  error
}

func (s *passwordResetUserRepositoryStub) FindByID(context.Context, uuid.UUID) (*entity.User, error) {
	return nil, entity.ErrUserNotFound
}
func (s *passwordResetUserRepositoryStub) FindByEmail(_ context.Context, email string) (*entity.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.user == nil || s.user.Email != email {
		return nil, entity.ErrUserNotFound
	}
	return s.user, nil
}
func (s *passwordResetUserRepositoryStub) Create(context.Context, *entity.User) (*entity.User, error) {
	return nil, entity.ErrInternal
}
func (s *passwordResetUserRepositoryStub) SoftDelete(context.Context, repository.ITransaction, uuid.UUID, time.Time) error {
	return entity.ErrInternal
}
func (s *passwordResetUserRepositoryStub) IncrementTokenVersion(context.Context, repository.ITransaction, uuid.UUID, time.Time) error {
	return entity.ErrInternal
}
func (s *passwordResetUserRepositoryStub) UpdateProfile(context.Context, uuid.UUID, string, string, time.Time) (*entity.User, error) {
	return nil, entity.ErrInternal
}

type passwordResetRepositoryStub struct {
	created      *entity.PasswordResetToken
	resetHash    string
	passwordHash string
	resetAt      time.Time
	resetErr     error
}

func (s *passwordResetRepositoryStub) Create(_ context.Context, token *entity.PasswordResetToken) error {
	s.created = token
	return nil
}
func (s *passwordResetRepositoryStub) ResetPassword(_ context.Context, tokenHash, passwordHash string, now time.Time) error {
	s.resetHash = tokenHash
	s.passwordHash = passwordHash
	s.resetAt = now
	return s.resetErr
}

type passwordResetHasherStub struct{ hashed string }

func (s passwordResetHasherStub) Hash(string) (string, error) { return s.hashed, nil }

type passwordResetMailerStub struct {
	email string
	token string
}

func (s *passwordResetMailerStub) SendPasswordReset(_ context.Context, email, token string) error {
	s.email = email
	s.token = token
	return nil
}

func TestPasswordResetRequestCreatesExpiringTokenAndSendsMail(t *testing.T) {
	now := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)
	user := &entity.User{ID: uuid.New(), Email: "user@example.com", Status: entity.UserStatusActive}
	users := &passwordResetUserRepositoryStub{user: user}
	tokens := &passwordResetRepositoryStub{}
	mailer := &passwordResetMailerStub{}
	uc := NewPasswordResetUsecase(users, tokens, passwordResetHasherStub{hashed: "hash"}, mailer, NewFixedClock(now))

	result, err := uc.Request(context.Background(), " USER@EXAMPLE.COM ")
	if err != nil {
		t.Fatalf("Request() error = %v", err)
	}
	if result == nil || result.Token == "" {
		t.Fatal("reset token must be returned to the controller")
	}
	if tokens.created == nil || tokens.created.UserID != user.ID {
		t.Fatalf("created token = %#v", tokens.created)
	}
	if !tokens.created.ExpiresAt.Equal(now.Add(passwordResetTTL)) {
		t.Fatalf("expiresAt = %v", tokens.created.ExpiresAt)
	}
	if mailer.email != user.Email || mailer.token != result.Token {
		t.Fatalf("mailer = email:%q token:%q", mailer.email, mailer.token)
	}
}

func TestPasswordResetRequestDoesNotRevealUnknownEmail(t *testing.T) {
	tokens := &passwordResetRepositoryStub{}
	mailer := &passwordResetMailerStub{}
	uc := NewPasswordResetUsecase(&passwordResetUserRepositoryStub{}, tokens, passwordResetHasherStub{hashed: "hash"}, mailer, NewFixedClock(time.Now()))

	result, err := uc.Request(context.Background(), "missing@example.com")
	if err != nil {
		t.Fatalf("Request() error = %v", err)
	}
	if result == nil || result.Token != "" {
		t.Fatalf("result = %#v", result)
	}
	if tokens.created != nil || mailer.email != "" {
		t.Fatal("unknown email must not create a token or send mail")
	}
}

func TestPasswordResetConfirmHashesTokenAndUpdatesPassword(t *testing.T) {
	now := time.Date(2026, 8, 8, 1, 0, 0, 0, time.UTC)
	tokens := &passwordResetRepositoryStub{}
	uc := NewPasswordResetUsecase(&passwordResetUserRepositoryStub{}, tokens, passwordResetHasherStub{hashed: "new-password-hash"}, nil, NewFixedClock(now))

	if err := uc.Confirm(context.Background(), "plain-token", "new-password"); err != nil {
		t.Fatalf("Confirm() error = %v", err)
	}
	sum := sha256.Sum256([]byte("plain-token"))
	if tokens.resetHash != hex.EncodeToString(sum[:]) {
		t.Fatalf("token hash = %q", tokens.resetHash)
	}
	if tokens.passwordHash != "new-password-hash" || !tokens.resetAt.Equal(now) {
		t.Fatalf("passwordHash=%q resetAt=%v", tokens.passwordHash, tokens.resetAt)
	}
}

func TestPasswordResetConfirmRejectsInvalidPasswordAndToken(t *testing.T) {
	uc := NewPasswordResetUsecase(&passwordResetUserRepositoryStub{}, &passwordResetRepositoryStub{}, passwordResetHasherStub{hashed: "hash"}, nil, NewFixedClock(time.Now()))
	if err := uc.Confirm(context.Background(), "", "new-password"); !errors.Is(err, entity.ErrInvalidPasswordResetToken) {
		t.Fatalf("empty token error = %v", err)
	}
	if err := uc.Confirm(context.Background(), "token", "short"); !errors.Is(err, entity.ErrInvalidInput) {
		t.Fatalf("short password error = %v", err)
	}
}

var _ repository.IUserRepository = (*passwordResetUserRepositoryStub)(nil)
var _ repository.IPasswordResetRepository = (*passwordResetRepositoryStub)(nil)
var _ IPasswordHasher = passwordResetHasherStub{}
var _ IPasswordResetMailer = (*passwordResetMailerStub)(nil)
