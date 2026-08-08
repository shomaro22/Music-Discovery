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

type IPasswordResetSystemLogWriter interface {
	CreateSystemLog(context.Context, *entity.SystemLog) error
}

type IPasswordResetUsecase interface {
	Request(ctx context.Context, email string) (*PasswordResetRequestResult, error)
	Confirm(ctx context.Context, token, newPassword string) error
}

type passwordResetUsecase struct {
	users         repository.IUserRepository
	tokens        repository.IPasswordResetRepository
	refreshTokens repository.IRefreshTokenRepository
	tx            repository.ITransactionManager
	hasher        IPasswordHasher
	mailer        IPasswordResetMailer
	clock         IClock
	systemLogs    IPasswordResetSystemLogWriter
}

func NewPasswordResetUsecase(
	users repository.IUserRepository,
	tokens repository.IPasswordResetRepository,
	refreshTokens repository.IRefreshTokenRepository,
	tx repository.ITransactionManager,
	hasher IPasswordHasher,
	mailer IPasswordResetMailer,
	clock IClock,
	systemLogs IPasswordResetSystemLogWriter,
) IPasswordResetUsecase {
	return &passwordResetUsecase{
		users: users, tokens: tokens, refreshTokens: refreshTokens, tx: tx,
		hasher: hasher, mailer: mailer, clock: clock, systemLogs: systemLogs,
	}
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
		if err := u.mailer.SendPasswordReset(ctx, user.Email, plain); err != nil {
			u.recordMailFailure(ctx, user.ID, err, now)
			// The public response remains indistinguishable from unknown-email requests.
			// The internal delivery failure is persisted to System Log for operations.
		}
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
	if u.tx == nil || u.refreshTokens == nil {
		return entity.ErrInternal
	}
	passwordHash, err := u.hasher.Hash(newPassword)
	if err != nil {
		return entity.ErrInternal
	}
	now := u.clock.Now()
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	tokenHash := hex.EncodeToString(sum[:])

	return u.tx.WithinTransaction(ctx, func(tx repository.ITransaction) error {
		resetToken, err := u.tokens.FindByHashForUpdate(ctx, tx, tokenHash)
		if err != nil {
			return err
		}
		if resetToken.UsedAt != nil || !now.Before(resetToken.ExpiresAt) {
			return entity.ErrInvalidPasswordResetToken
		}
		if err := u.users.UpdatePassword(ctx, tx, resetToken.UserID, passwordHash, now); err != nil {
			if errors.Is(err, entity.ErrUserNotFound) {
				return entity.ErrInvalidPasswordResetToken
			}
			return err
		}
		if err := u.users.IncrementTokenVersion(ctx, tx, resetToken.UserID, now); err != nil {
			return err
		}
		if err := u.refreshTokens.RevokeAllByUserID(ctx, tx, resetToken.UserID, now); err != nil {
			return err
		}
		return u.tokens.MarkUsed(ctx, tx, resetToken.ID, now)
	})
}

func (u *passwordResetUsecase) recordMailFailure(ctx context.Context, userID uuid.UUID, sendErr error, now time.Time) {
	if u.systemLogs == nil {
		return
	}
	_ = u.systemLogs.CreateSystemLog(ctx, &entity.SystemLog{
		ID: uuid.New(), Level: "ERROR", Category: "application", Event: "password_reset_mail_failed",
		ErrorCode: "PASSWORD_RESET_MAIL_FAILED", Message: "パスワード再設定メールの送信に失敗しました",
		UserID: &userID, ExternalService: "smtp", ErrorDetail: sendErr.Error(), CreatedAt: now,
	})
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
