package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/repository"
	"music-discovery/backend/validator"

	"github.com/google/uuid"
)

type withdrawUserRepositoryStub struct {
	user            *entity.User
	findErr         error
	softDeleteErr   error
	incrementErr    error
	softDeleteCalls int
	incrementCalls  int
	softDeleteAt    time.Time
	incrementAt     time.Time
}

func (s *withdrawUserRepositoryStub) FindByID(context.Context, uuid.UUID) (*entity.User, error) {
	return s.user, s.findErr
}
func (s *withdrawUserRepositoryStub) FindByEmail(context.Context, string) (*entity.User, error) {
	return nil, entity.ErrUserNotFound
}
func (s *withdrawUserRepositoryStub) Create(context.Context, *entity.User) (*entity.User, error) {
	return nil, entity.ErrInternal
}
func (s *withdrawUserRepositoryStub) SoftDelete(_ context.Context, _ repository.ITransaction, _ uuid.UUID, now time.Time) error {
	s.softDeleteCalls++
	s.softDeleteAt = now
	return s.softDeleteErr
}
func (s *withdrawUserRepositoryStub) IncrementTokenVersion(_ context.Context, _ repository.ITransaction, _ uuid.UUID, now time.Time) error {
	s.incrementCalls++
	s.incrementAt = now
	return s.incrementErr
}

type withdrawRefreshRepositoryStub struct {
	revokeErr   error
	revokeCalls int
	revokeAt    time.Time
}

func (s *withdrawRefreshRepositoryStub) Create(context.Context, *entity.RefreshToken) error {
	return nil
}
func (s *withdrawRefreshRepositoryStub) FindByTokenHash(context.Context, string) (*entity.RefreshToken, error) {
	return nil, entity.ErrInvalidRefreshToken
}
func (s *withdrawRefreshRepositoryStub) Rotate(context.Context, uuid.UUID, *entity.RefreshToken, time.Time) error {
	return nil
}
func (s *withdrawRefreshRepositoryStub) RevokeFamily(context.Context, uuid.UUID, time.Time) error {
	return nil
}
func (s *withdrawRefreshRepositoryStub) RevokeAllByUserID(_ context.Context, _ repository.ITransaction, _ uuid.UUID, now time.Time) error {
	s.revokeCalls++
	s.revokeAt = now
	return s.revokeErr
}

type withdrawTransactionManagerStub struct {
	beginErr error
	fnErr    error
	calls    int
}

func (s *withdrawTransactionManagerStub) CreateUserAndRefreshToken(context.Context, *entity.User, *entity.RefreshToken) error {
	return nil
}
func (s *withdrawTransactionManagerStub) RevokeFamilyAndIncrementTokenVersion(context.Context, uuid.UUID, uuid.UUID, time.Time) error {
	return nil
}
func (s *withdrawTransactionManagerStub) WithinTransaction(ctx context.Context, fn func(repository.ITransaction) error) error {
	s.calls++
	if s.beginErr != nil {
		return s.beginErr
	}
	if err := fn(nil); err != nil {
		return err
	}
	return s.fnErr
}

type withdrawTokenServiceStub struct{ compareErr error }

func (s *withdrawTokenServiceStub) ComparePassword(string, string) error { return s.compareErr }
func (s *withdrawTokenServiceStub) GenerateAccessToken(*entity.User, time.Time) (string, int64, error) {
	return "", 0, nil
}
func (s *withdrawTokenServiceStub) GenerateRefreshToken() (string, string, error) { return "", "", nil }
func (s *withdrawTokenServiceStub) HashRefreshToken(string) string                { return "" }
func (s *withdrawTokenServiceStub) GenerateCSRFToken() (string, error)            { return "", nil }
func (s *withdrawTokenServiceStub) GenerateFamilyID() uuid.UUID                   { return uuid.New() }
func (s *withdrawTokenServiceStub) RefreshTokenTTL() time.Duration                { return time.Hour }
func (s *withdrawTokenServiceStub) VerifyAccessToken(string, time.Time) (uuid.UUID, entity.Role, uint64, error) {
	return uuid.Nil, entity.RoleUser, 0, nil
}

func newWithdrawUsecaseForTest(
	userRepo *withdrawUserRepositoryStub,
	refreshRepo *withdrawRefreshRepositoryStub,
	txManager *withdrawTransactionManagerStub,
	tokenService *withdrawTokenServiceStub,
	now time.Time,
) IUserUsecase {
	return NewUserUsecase(
		userRepo,
		validator.NewUserValidator(),
		refreshRepo,
		txManager,
		&fakeHasher{},
		tokenService,
		&fakeClock{now: now},
	)
}

func activeWithdrawUser() *entity.User {
	return &entity.User{
		ID:           uuid.New(),
		PasswordHash: "hashed-password",
		Role:         entity.RoleUser,
		Status:       entity.UserStatusActive,
	}
}

func TestWithdrawSuccessUsesOneTimestamp(t *testing.T) {
	now := time.Date(2026, 7, 23, 1, 0, 0, 0, time.UTC)
	user := activeWithdrawUser()
	userRepo := &withdrawUserRepositoryStub{user: user}
	refreshRepo := &withdrawRefreshRepositoryStub{}
	txManager := &withdrawTransactionManagerStub{}

	result, err := newWithdrawUsecaseForTest(userRepo, refreshRepo, txManager, &withdrawTokenServiceStub{}, now).
		Withdraw(context.Background(), user.ID, WithdrawInput{CurrentPassword: "password"})

	if err != nil {
		t.Fatalf("Withdraw() error = %v", err)
	}
	if result == nil || result.Message == "" {
		t.Fatal("expected success result")
	}
	if userRepo.softDeleteCalls != 1 || userRepo.incrementCalls != 1 || refreshRepo.revokeCalls != 1 {
		t.Fatalf("unexpected calls: delete=%d increment=%d revoke=%d", userRepo.softDeleteCalls, userRepo.incrementCalls, refreshRepo.revokeCalls)
	}
	if userRepo.softDeleteAt != now || userRepo.incrementAt != now || refreshRepo.revokeAt != now {
		t.Fatal("withdraw operations must use the same timestamp")
	}
}

func TestWithdrawRejectsEmptyPasswordBeforeRepositoryAccess(t *testing.T) {
	userRepo := &withdrawUserRepositoryStub{}
	txManager := &withdrawTransactionManagerStub{}
	_, err := newWithdrawUsecaseForTest(userRepo, &withdrawRefreshRepositoryStub{}, txManager, &withdrawTokenServiceStub{}, time.Now().UTC()).
		Withdraw(context.Background(), uuid.New(), WithdrawInput{})

	if !errors.Is(err, entity.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if txManager.calls != 0 {
		t.Fatal("transaction must not start")
	}
}

func TestWithdrawRejectsPasswordMismatchWithoutTransaction(t *testing.T) {
	user := activeWithdrawUser()
	txManager := &withdrawTransactionManagerStub{}
	_, err := newWithdrawUsecaseForTest(
		&withdrawUserRepositoryStub{user: user},
		&withdrawRefreshRepositoryStub{},
		txManager,
		&withdrawTokenServiceStub{compareErr: entity.ErrInvalidCredentials},
		time.Now().UTC(),
	).Withdraw(context.Background(), user.ID, WithdrawInput{CurrentPassword: "wrong"})

	if !errors.Is(err, entity.ErrCurrentPasswordMismatch) {
		t.Fatalf("expected ErrCurrentPasswordMismatch, got %v", err)
	}
	if txManager.calls != 0 {
		t.Fatal("transaction must not start")
	}
}

func TestWithdrawMapsMissingAndAlreadyWithdrawnToInvalidCredentials(t *testing.T) {
	for _, tc := range []struct {
		name          string
		findErr       error
		softDeleteErr error
	}{
		{name: "missing before transaction", findErr: entity.ErrUserNotFound},
		{name: "already withdrawn during transaction", softDeleteErr: entity.ErrUserAlreadyWithdrawn},
	} {
		t.Run(tc.name, func(t *testing.T) {
			user := activeWithdrawUser()
			userRepo := &withdrawUserRepositoryStub{user: user, findErr: tc.findErr, softDeleteErr: tc.softDeleteErr}
			_, err := newWithdrawUsecaseForTest(userRepo, &withdrawRefreshRepositoryStub{}, &withdrawTransactionManagerStub{}, &withdrawTokenServiceStub{}, time.Now().UTC()).
				Withdraw(context.Background(), user.ID, WithdrawInput{CurrentPassword: "password"})
			if !errors.Is(err, entity.ErrInvalidCredentials) {
				t.Fatalf("expected ErrInvalidCredentials, got %v", err)
			}
		})
	}
}

func TestWithdrawRejectsNonActiveUser(t *testing.T) {
	user := activeWithdrawUser()
	user.Status = entity.UserStatusSuspended
	txManager := &withdrawTransactionManagerStub{}
	_, err := newWithdrawUsecaseForTest(&withdrawUserRepositoryStub{user: user}, &withdrawRefreshRepositoryStub{}, txManager, &withdrawTokenServiceStub{}, time.Now().UTC()).
		Withdraw(context.Background(), user.ID, WithdrawInput{CurrentPassword: "password"})

	if !errors.Is(err, entity.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	if txManager.calls != 0 {
		t.Fatal("transaction must not start")
	}
}

func TestWithdrawMapsTransactionFailuresToInternalError(t *testing.T) {
	for _, tc := range []struct {
		name         string
		deleteErr    error
		incrementErr error
		revokeErr    error
		commitErr    error
	}{
		{name: "soft delete", deleteErr: entity.ErrInternal},
		{name: "token version", incrementErr: entity.ErrInternal},
		{name: "refresh revoke", revokeErr: entity.ErrInternal},
		{name: "commit", commitErr: entity.ErrInternal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			user := activeWithdrawUser()
			_, err := newWithdrawUsecaseForTest(
				&withdrawUserRepositoryStub{user: user, softDeleteErr: tc.deleteErr, incrementErr: tc.incrementErr},
				&withdrawRefreshRepositoryStub{revokeErr: tc.revokeErr},
				&withdrawTransactionManagerStub{fnErr: tc.commitErr},
				&withdrawTokenServiceStub{},
				time.Now().UTC(),
			).Withdraw(context.Background(), user.ID, WithdrawInput{CurrentPassword: "password"})
			if !errors.Is(err, entity.ErrInternal) {
				t.Fatalf("expected ErrInternal, got %v", err)
			}
		})
	}
}

func (s *withdrawUserRepositoryStub) UpdateProfile(_ context.Context, _ uuid.UUID, _, _ string, _ time.Time) (*entity.User, error) {
	return s.user, nil
}

var _ repository.IUserRepository = (*withdrawUserRepositoryStub)(nil)
var _ repository.IRefreshTokenRepository = (*withdrawRefreshRepositoryStub)(nil)
var _ repository.IAuthTransactionManager = (*withdrawTransactionManagerStub)(nil)
var _ ITokenService = (*withdrawTokenServiceStub)(nil)
