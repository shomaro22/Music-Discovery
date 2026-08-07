package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/repository"
	"music-discovery/backend/validator"

	"github.com/google/uuid"
)

const passwordResetTTL = 30 * time.Minute

type PasswordResetRequestResult struct{ Token string }

type IPasswordResetMailer interface {
	SendPasswordReset(ctx context.Context, email, token string) error
}

type IPasswordResetUsecase interface {
	Request(ctx context.Context, email string) (*PasswordResetRequestResult, error)
	Confirm(ctx context.Context, token, newPassword string) error
}

type passwordResetUsecase struct {
	users  repository.IUserRepository
	tokens repository.IPasswordResetRepository
	hasher IPasswordHasher
	mailer IPasswordResetMailer
	clock  IClock
}

func NewPasswordResetUsecase(users repository.IUserRepository, tokens repository.IPasswordResetRepository, hasher IPasswordHasher, mailer IPasswordResetMailer, clock IClock) IPasswordResetUsecase {
	return &passwordResetUsecase{users: users, tokens: tokens, hasher: hasher, mailer: mailer, clock: clock}
}

func (u *passwordResetUsecase) Request(ctx context.Context, email string) (*PasswordResetRequestResult, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	address, err := mail.ParseAddress(normalized)
	if err != nil || address.Address != normalized || len(normalized) > 254 {
		return nil, entity.ErrInvalidInput
	}
	user, err := u.users.FindByEmail(ctx, normalized)
	if errors.Is(err, entity.ErrUserNotFound) {
		return &PasswordResetRequestResult{}, nil
	}
	if err != nil {
		return nil, entity.ErrInternal
	}
	if user.Status != entity.UserStatusActive || user.DeletedAt.Valid {
		return &PasswordResetRequestResult{}, nil
	}
	plain, hash, err := newPasswordResetToken()
	if err != nil {
		return nil, entity.ErrInternal
	}
	now := u.clock.Now()
	if err := u.tokens.Create(ctx, &entity.PasswordResetToken{ID: uuid.New(), UserID: user.ID, TokenHash: hash, ExpiresAt: now.Add(passwordResetTTL), CreatedAt: now}); err != nil {
		return nil, err
	}
	if u.mailer != nil {
		_ = u.mailer.SendPasswordReset(ctx, user.Email, plain)
	}
	return &PasswordResetRequestResult{Token: plain}, nil
}

func (u *passwordResetUsecase) Confirm(ctx context.Context, token, newPassword string) error {
	if strings.TrimSpace(token) == "" {
		return entity.ErrInvalidPasswordResetToken
	}
	_, validationErrors := validator.ValidateSignUp("reset", "reset@example.com", newPassword)
	if _, invalid := validationErrors["password"]; invalid {
		return entity.ErrInvalidInput
	}
	passwordHash, err := u.hasher.Hash(newPassword)
	if err != nil {
		return entity.ErrInternal
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return u.tokens.ResetPassword(ctx, hex.EncodeToString(sum[:]), passwordHash, u.clock.Now())
}

func newPasswordResetToken() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	plain := base64.RawURLEncoding.EncodeToString(bytes)
	sum := sha256.Sum256([]byte(plain))
	return plain, hex.EncodeToString(sum[:]), nil
}
