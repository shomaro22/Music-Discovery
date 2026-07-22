package usecase

import (
	"context"
	"errors"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/repository"
	"music-discovery/backend/validator"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type SignUpInput struct{ Name, Email, Password string }
type LoginInput struct{ Email, Password string }
type WithdrawInput struct{ CurrentPassword string }
type AuthResult struct {
	AccessToken           string
	ExpiresIn             int64
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
	CSRFToken             string
	User                  *entity.User
}

type LoginResult = AuthResult
type SignUpResult = AuthResult

type WithdrawResult struct{ Message string }

type CurrentUserResult struct {
	ID     uuid.UUID
	Name   string
	Email  string
	Role   entity.UserRole
	Status entity.UserStatus
}

type RefreshResult struct {
	AccessToken           string
	ExpiresIn             int64
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
	CSRFToken             string
}

type IPasswordHasher interface {
	Hash(password string) (string, error)
}
type bcryptPasswordHasher struct{}

func NewPasswordHasher() IPasswordHasher { return &bcryptPasswordHasher{} }
func (h *bcryptPasswordHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", entity.ErrInternal
	}
	return string(hash), nil
}

type IClock interface{ Now() time.Time }
type systemClock struct{}

func NewClock() IClock                { return &systemClock{} }
func (c *systemClock) Now() time.Time { return time.Now().UTC() }

type IUserUsecase interface {
	SignUp(ctx context.Context, input SignUpInput) (*SignUpResult, error)
	Login(ctx context.Context, input LoginInput) (*LoginResult, error)
	Refresh(ctx context.Context, refreshToken string) (*RefreshResult, error)
	Logout(ctx context.Context, refreshToken string) error
	CurrentUser(ctx context.Context, userID uuid.UUID) (*CurrentUserResult, error)
	ValidateTokenVersion(ctx context.Context, userID uuid.UUID, tokenVersion uint64) (*entity.User, error)
	Withdraw(ctx context.Context, userID uuid.UUID, input WithdrawInput) (*WithdrawResult, error)
}

type userUsecase struct {
	userRepository         repository.IUserRepository
	userValidator          validator.IUserValidator
	refreshTokenRepository repository.IRefreshTokenRepository
	authTransactionManager repository.IAuthTransactionManager
	passwordHasher         IPasswordHasher
	tokenService           ITokenService
	clock                  IClock
}

func NewUserUsecase(
	userRepository repository.IUserRepository,
	userValidator validator.IUserValidator,
	refreshTokenRepository repository.IRefreshTokenRepository,
	authTransactionManager repository.IAuthTransactionManager,
	passwordHasher IPasswordHasher,
	tokenService ITokenService,
	clock IClock,
) IUserUsecase {
	return &userUsecase{userRepository: userRepository, userValidator: userValidator, refreshTokenRepository: refreshTokenRepository, authTransactionManager: authTransactionManager, passwordHasher: passwordHasher, tokenService: tokenService, clock: clock}
}

func (u *userUsecase) SignUp(ctx context.Context, input SignUpInput) (*SignUpResult, error) {
	values, validationErrors := u.userValidator.ValidateSignUp(input.Name, input.Email, input.Password)
	if len(validationErrors) > 0 {
		return nil, entity.ErrInvalidInput
	}
	input = SignUpInput{Name: values.Name, Email: values.Email, Password: values.Password}
	_, err := u.userRepository.FindByEmail(ctx, input.Email)
	switch {
	case err == nil:
		return nil, entity.ErrEmailAlreadyExists
	case errors.Is(err, entity.ErrUserNotFound):
	default:
		return nil, entity.ErrInternal
	}
	passwordHash, err := u.passwordHasher.Hash(input.Password)
	if err != nil {
		return nil, entity.ErrInternal
	}
	now := u.clock.Now()
	user := &entity.User{
		ID: uuid.New(), Name: input.Name, Email: input.Email, PasswordHash: passwordHash,
		Role: entity.UserRoleUser, Status: entity.UserStatusActive, TokenVersion: 0,
		CreatedAt: now, UpdatedAt: now,
	}
	authResult, refreshToken, err := u.buildAuthResult(user, now)
	if err != nil {
		return nil, err
	}
	if err := u.authTransactionManager.CreateUserAndRefreshToken(ctx, user, refreshToken); err != nil {
		switch {
		case errors.Is(err, entity.ErrEmailAlreadyExists):
			return nil, entity.ErrEmailAlreadyExists
		default:
			return nil, entity.ErrInternal
		}
	}

	return authResult, nil
}

func (u *userUsecase) buildAuthResult(user *entity.User, now time.Time) (*AuthResult, *entity.RefreshToken, error) {
	access, expiresIn, err := u.tokenService.GenerateAccessToken(user, now)
	if err != nil {
		return nil, nil, entity.ErrInternal
	}
	plain, hash, err := u.tokenService.GenerateRefreshToken()
	if err != nil {
		return nil, nil, entity.ErrInternal
	}
	csrf, err := u.tokenService.GenerateCSRFToken()
	if err != nil {
		return nil, nil, entity.ErrInternal
	}
	expiresAt := now.Add(u.tokenService.RefreshTokenTTL())
	refresh := &entity.RefreshToken{
		ID: uuid.New(), UserID: user.ID, TokenHash: hash, FamilyID: u.tokenService.GenerateFamilyID(),
		TokenVersion: user.TokenVersion, ExpiresAt: expiresAt, CreatedAt: now, UpdatedAt: now,
	}
	result := &AuthResult{
		AccessToken: access, ExpiresIn: expiresIn, RefreshToken: plain,
		RefreshTokenExpiresAt: expiresAt, CSRFToken: csrf, User: user,
	}
	return result, refresh, nil
}

func (u *userUsecase) issueAuthResult(ctx context.Context, user *entity.User, now time.Time) (*AuthResult, error) {
	result, refresh, err := u.buildAuthResult(user, now)
	if err != nil {
		return nil, err
	}
	if err := u.refreshTokenRepository.Create(ctx, refresh); err != nil {
		return nil, entity.ErrInternal
	}
	return result, nil
}

const dummyPasswordHash = "$2a$10$7EqJtq98hPqEX7fNZaFWoO5LQzYOgTBz8tLJ7YQnbvCw8wq9a3E0K"

func (u *userUsecase) Login(ctx context.Context, input LoginInput) (*LoginResult, error) {
	values, validationErrors := u.userValidator.ValidateLogin(input.Email, input.Password)
	if len(validationErrors) > 0 {
		return nil, entity.ErrInvalidInput
	}
	input = LoginInput{Email: values.Email, Password: values.Password}
	user, err := u.userRepository.FindByEmail(ctx, input.Email)
	if errors.Is(err, entity.ErrUserNotFound) {
		_ = u.tokenService.ComparePassword(dummyPasswordHash, input.Password)
		return nil, entity.ErrInvalidCredentials
	}
	if err != nil {
		return nil, entity.ErrInternal
	}
	if err := u.tokenService.ComparePassword(user.PasswordHash, input.Password); err != nil {
		return nil, entity.ErrInvalidCredentials
	}
	if user.Status == entity.UserStatusSuspended {
		return nil, entity.ErrUserSuspended
	}
	if user.Status != entity.UserStatusActive {
		return nil, entity.ErrInvalidCredentials
	}
	result, err := u.issueAuthResult(ctx, user, u.clock.Now())
	if err != nil {
		return nil, err
	}
	return (*LoginResult)(result), nil
}

func (u *userUsecase) Refresh(ctx context.Context, plainToken string) (*RefreshResult, error) {
	if plainToken == "" {
		return nil, entity.ErrInvalidRefreshToken
	}
	now := u.clock.Now()
	current, err := u.refreshTokenRepository.FindByTokenHash(ctx, u.tokenService.HashRefreshToken(plainToken))
	if err != nil {
		if errors.Is(err, entity.ErrInvalidRefreshToken) {
			return nil, entity.ErrInvalidRefreshToken
		}
		return nil, entity.ErrInternal
	}
	if current.IsRevoked() || current.IsUsed() {
		if err := u.authTransactionManager.RevokeFamilyAndIncrementTokenVersion(ctx, current.FamilyID, current.UserID, now); err != nil {
			return nil, entity.ErrInternal
		}
		return nil, entity.ErrRefreshTokenReused
	}
	if current.IsExpired(now) {
		return nil, entity.ErrInvalidRefreshToken
	}
	user, err := u.userRepository.FindByID(ctx, current.UserID)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			return nil, entity.ErrInvalidRefreshToken
		}
		return nil, entity.ErrInternal
	}
	if current.TokenVersion != user.TokenVersion {
		return nil, entity.ErrInvalidRefreshToken
	}
	if user.Status == entity.UserStatusSuspended {
		return nil, entity.ErrUserSuspended
	}
	if user.Status != entity.UserStatusActive {
		return nil, entity.ErrInvalidRefreshToken
	}
	newPlain, newHash, err := u.tokenService.GenerateRefreshToken()
	if err != nil {
		return nil, entity.ErrInternal
	}
	expiresAt := now.Add(u.tokenService.RefreshTokenTTL())
	next := &entity.RefreshToken{
		ID: uuid.New(), UserID: user.ID, TokenHash: newHash, FamilyID: current.FamilyID,
		TokenVersion: user.TokenVersion, ExpiresAt: expiresAt, CreatedAt: now, UpdatedAt: now,
	}
	if err := u.refreshTokenRepository.Rotate(ctx, current.ID, next, now); err != nil {
		if errors.Is(err, entity.ErrRefreshTokenReused) {
			if revokeErr := u.authTransactionManager.RevokeFamilyAndIncrementTokenVersion(ctx, current.FamilyID, current.UserID, now); revokeErr != nil {
				return nil, entity.ErrInternal
			}
			return nil, entity.ErrRefreshTokenReused
		}
		return nil, entity.ErrInternal
	}
	accessToken, expiresIn, err := u.tokenService.GenerateAccessToken(user, now)
	if err != nil {
		return nil, entity.ErrInternal
	}
	csrfToken, err := u.tokenService.GenerateCSRFToken()
	if err != nil {
		return nil, entity.ErrInternal
	}
	return &RefreshResult{
		AccessToken: accessToken, ExpiresIn: expiresIn, RefreshToken: newPlain,
		RefreshTokenExpiresAt: expiresAt, CSRFToken: csrfToken,
	}, nil
}

func (u *userUsecase) Logout(ctx context.Context, plainToken string) error {
	if plainToken == "" {
		return nil
	}
	token, err := u.refreshTokenRepository.FindByTokenHash(ctx, u.tokenService.HashRefreshToken(plainToken))
	if err != nil {
		if errors.Is(err, entity.ErrInvalidRefreshToken) {
			return nil
		}
		return entity.ErrInternal
	}
	if token == nil || token.FamilyID == uuid.Nil {
		return nil
	}
	if err := u.refreshTokenRepository.RevokeFamily(ctx, token.FamilyID, u.clock.Now()); err != nil {
		return entity.ErrInternal
	}
	return nil
}

func (u *userUsecase) CurrentUser(ctx context.Context, userID uuid.UUID) (*CurrentUserResult, error) {
	if userID == uuid.Nil {
		return nil, entity.ErrInvalidCredentials
	}

	user, err := u.userRepository.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			return nil, entity.ErrInvalidCredentials
		}
		return nil, entity.ErrInternal
	}
	if user.Status == entity.UserStatusSuspended {
		return nil, entity.ErrUserSuspended
	}
	if user.Status != entity.UserStatusActive {
		return nil, entity.ErrInvalidCredentials
	}

	return &CurrentUserResult{
		ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role, Status: user.Status,
	}, nil
}

func (u *userUsecase) ValidateTokenVersion(ctx context.Context, userID uuid.UUID, tokenVersion uint64) (*entity.User, error) {
	user, err := u.userRepository.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.Status == entity.UserStatusSuspended {
		return nil, entity.ErrUserSuspended
	}
	if user.Status != entity.UserStatusActive || user.TokenVersion != tokenVersion {
		return nil, entity.ErrInvalidCredentials
	}
	return user, nil
}

func (u *userUsecase) Withdraw(ctx context.Context, userID uuid.UUID, input WithdrawInput) (*WithdrawResult, error) {
	if userID == uuid.Nil || input.CurrentPassword == "" {
		return nil, entity.ErrInvalidInput
	}
	user, err := u.userRepository.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			return nil, entity.ErrInvalidCredentials
		}
		return nil, entity.ErrInternal
	}
	if user.Status != entity.UserStatusActive {
		return nil, entity.ErrInvalidCredentials
	}
	if err := u.tokenService.ComparePassword(user.PasswordHash, input.CurrentPassword); err != nil {
		return nil, entity.ErrCurrentPasswordMismatch
	}

	now := u.clock.Now()
	if err := u.authTransactionManager.WithinTransaction(
		ctx,
		func(tx repository.ITransaction) error {
			if err := u.userRepository.SoftDelete(ctx, tx, userID, now); err != nil {
				return err
			}
			if err := u.userRepository.IncrementTokenVersion(ctx, tx, userID, now); err != nil {
				return err
			}
			if err := u.refreshTokenRepository.RevokeAllByUserID(ctx, tx, userID, now); err != nil {
				return err
			}
			return nil
		}); err != nil {
		switch {
		case errors.Is(err, entity.ErrUserNotFound),
			errors.Is(err, entity.ErrUserAlreadyWithdrawn):
			return nil, entity.ErrInvalidCredentials
		default:
			return nil, entity.ErrInternal
		}
	}
	return &WithdrawResult{Message: "退会が完了しました。"}, nil
}
