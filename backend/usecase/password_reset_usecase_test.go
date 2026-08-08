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
	user              *entity.User
	err               error
	updatedUserID     uuid.UUID
	updatedPassword   string
	incrementedUserID uuid.UUID
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
func (s *passwordResetUserRepositoryStub) IncrementTokenVersion(_ context.Context, _ repository.ITransaction, userID uuid.UUID, _ time.Time) error {
	s.incrementedUserID = userID
	return nil
}
func (s *passwordResetUserRepositoryStub) UpdatePassword(_ context.Context, _ repository.ITransaction, userID uuid.UUID, passwordHash string, _ time.Time) error {
	s.updatedUserID = userID
	s.updatedPassword = passwordHash
	return nil
}
func (s *passwordResetUserRepositoryStub) UpdateProfile(context.Context, uuid.UUID, string, string, time.Time) (*entity.User, error) {
	return nil, entity.ErrInternal
}

type passwordResetRepositoryStub struct {
	created  *entity.PasswordResetToken
	found    *entity.PasswordResetToken
	findHash string
	markUsed uuid.UUID
	resetErr error
}

func (s *passwordResetRepositoryStub) Create(_ context.Context, token *entity.PasswordResetToken) error {
	s.created = token
	return nil
}
func (s *passwordResetRepositoryStub) FindByHashForUpdate(_ context.Context, _ repository.ITransaction, tokenHash string) (*entity.PasswordResetToken, error) {
	s.findHash = tokenHash
	if s.resetErr != nil {
		return nil, s.resetErr
	}
	if s.found == nil {
		return nil, entity.ErrInvalidPasswordResetToken
	}
	return s.found, nil
}
func (s *passwordResetRepositoryStub) MarkUsed(_ context.Context, _ repository.ITransaction, tokenID uuid.UUID, _ time.Time) error {
	s.markUsed = tokenID
	return nil
}

type passwordResetRefreshRepositoryStub struct{ revokedUserID uuid.UUID }

func (s *passwordResetRefreshRepositoryStub) Create(context.Context, *entity.RefreshToken) error {
	return nil
}
func (s *passwordResetRefreshRepositoryStub) FindByTokenHash(context.Context, string) (*entity.RefreshToken, error) {
	return nil, entity.ErrInvalidRefreshToken
}
func (s *passwordResetRefreshRepositoryStub) Rotate(context.Context, uuid.UUID, *entity.RefreshToken, time.Time) error {
	return nil
}
func (s *passwordResetRefreshRepositoryStub) RevokeFamily(context.Context, uuid.UUID, time.Time) error {
	return nil
}
func (s *passwordResetRefreshRepositoryStub) RevokeAllByUserID(_ context.Context, _ repository.ITransaction, userID uuid.UUID, _ time.Time) error {
	s.revokedUserID = userID
	return nil
}

type passwordResetTransactionStub struct{ err error }

func (s passwordResetTransactionStub) WithinTransaction(_ context.Context, fn func(repository.ITransaction) error) error {
	if s.err != nil {
		return s.err
	}
	return fn(nil)
}

type passwordResetHasherStub struct{ hashed string }

func (s passwordResetHasherStub) Hash(string) (string, error) { return s.hashed, nil }

type passwordResetMailerStub struct {
	email, token string
	err          error
}

func (s *passwordResetMailerStub) SendPasswordReset(_ context.Context, email, token string) error {
	s.email, s.token = email, token
	return s.err
}

type passwordResetSystemLogStub struct{ log *entity.SystemLog }

func (s *passwordResetSystemLogStub) CreateSystemLog(_ context.Context, value *entity.SystemLog) error {
	s.log = value
	return nil
}

func newPasswordResetUC(users repository.IUserRepository, tokens repository.IPasswordResetRepository, mailer IPasswordResetMailer, clock IClock, logs IPasswordResetSystemLogWriter) (IPasswordResetUsecase, *passwordResetRefreshRepositoryStub) {
	refresh := &passwordResetRefreshRepositoryStub{}
	return NewPasswordResetUsecase(users, tokens, refresh, passwordResetTransactionStub{}, passwordResetHasherStub{hashed: "new-password-hash"}, mailer, clock, logs), refresh
}

func TestPasswordResetRequestCreatesExpiringTokenAndSendsMail(t *testing.T) {
	now := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)
	user := &entity.User{ID: uuid.New(), Email: "user@example.com", Status: entity.UserStatusActive}
	users := &passwordResetUserRepositoryStub{user: user}
	tokens := &passwordResetRepositoryStub{}
	mailer := &passwordResetMailerStub{}
	uc, _ := newPasswordResetUC(users, tokens, mailer, NewFixedClock(now), nil)

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
	uc, _ := newPasswordResetUC(&passwordResetUserRepositoryStub{}, tokens, mailer, NewFixedClock(time.Now()), nil)
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

func TestPasswordResetRequestLogsMailFailureButKeepsGenericSuccess(t *testing.T) {
	now := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)
	user := &entity.User{ID: uuid.New(), Email: "user@example.com", Status: entity.UserStatusActive}
	logs := &passwordResetSystemLogStub{}
	uc, _ := newPasswordResetUC(&passwordResetUserRepositoryStub{user: user}, &passwordResetRepositoryStub{}, &passwordResetMailerStub{err: errors.New("smtp down")}, NewFixedClock(now), logs)
	result, err := uc.Request(context.Background(), user.Email)
	if err != nil {
		t.Fatalf("Request() error = %v", err)
	}
	if result == nil || result.Token == "" {
		t.Fatal("token should still be created")
	}
	if logs.log == nil || logs.log.Event != "password_reset_mail_failed" || logs.log.UserID == nil || *logs.log.UserID != user.ID {
		t.Fatalf("system log = %#v", logs.log)
	}
}

func TestPasswordResetConfirmOwnsTransactionalFlow(t *testing.T) {
	now := time.Date(2026, 8, 8, 1, 0, 0, 0, time.UTC)
	userID := uuid.New()
	resetID := uuid.New()
	tokens := &passwordResetRepositoryStub{found: &entity.PasswordResetToken{ID: resetID, UserID: userID, ExpiresAt: now.Add(passwordResetTTL)}}
	users := &passwordResetUserRepositoryStub{}
	uc, refresh := newPasswordResetUC(users, tokens, nil, NewFixedClock(now), nil)

	if err := uc.Confirm(context.Background(), "plain-token", "new-password"); err != nil {
		t.Fatalf("Confirm() error = %v", err)
	}
	sum := sha256.Sum256([]byte("plain-token"))
	if tokens.findHash != hex.EncodeToString(sum[:]) {
		t.Fatalf("token hash = %q", tokens.findHash)
	}
	if users.updatedUserID != userID || users.updatedPassword != "new-password-hash" {
		t.Fatalf("password update user=%v hash=%q", users.updatedUserID, users.updatedPassword)
	}
	if users.incrementedUserID != userID {
		t.Fatalf("token version user=%v", users.incrementedUserID)
	}
	if refresh.revokedUserID != userID {
		t.Fatalf("refresh revoke user=%v", refresh.revokedUserID)
	}
	if tokens.markUsed != resetID {
		t.Fatalf("marked token=%v", tokens.markUsed)
	}
}

func TestPasswordResetConfirmRejectsInvalidPasswordAndToken(t *testing.T) {
	uc, _ := newPasswordResetUC(&passwordResetUserRepositoryStub{}, &passwordResetRepositoryStub{}, nil, NewFixedClock(time.Now()), nil)
	if err := uc.Confirm(context.Background(), "", "new-password"); !errors.Is(err, entity.ErrInvalidPasswordResetToken) {
		t.Fatalf("empty token error = %v", err)
	}
	if err := uc.Confirm(context.Background(), "token", "short"); !errors.Is(err, entity.ErrInvalidInput) {
		t.Fatalf("short password error = %v", err)
	}
}

var _ repository.IUserRepository = (*passwordResetUserRepositoryStub)(nil)
var _ repository.IPasswordResetRepository = (*passwordResetRepositoryStub)(nil)
var _ repository.IRefreshTokenRepository = (*passwordResetRefreshRepositoryStub)(nil)
var _ repository.ITransactionManager = passwordResetTransactionStub{}
var _ IPasswordHasher = passwordResetHasherStub{}
var _ IPasswordResetMailer = (*passwordResetMailerStub)(nil)
var _ IPasswordResetSystemLogWriter = (*passwordResetSystemLogStub)(nil)
