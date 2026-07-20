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
type LoginResult struct {
	AccessToken           string
	ExpiresIn             int64
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
	CSRFToken             string
	User                  *entity.User
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
	SignUp(ctx context.Context, input SignUpInput) (*entity.User, error)
	Login(ctx context.Context, input LoginInput) (*LoginResult, error)
	Refresh(ctx context.Context, refreshToken string) (*RefreshResult, error)
	ValidateTokenVersion(ctx context.Context, userID uuid.UUID, tokenVersion uint64) (*entity.User, error)
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

func (u *userUsecase) SignUp(ctx context.Context, input SignUpInput) (*entity.User, error) {
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
	user := &entity.User{Name: input.Name, Email: input.Email, PasswordHash: passwordHash, Role: entity.UserRoleUser, Status: entity.UserStatusActive, TokenVersion: 0, CreatedAt: now, UpdatedAt: now}
	created, err := u.userRepository.Create(ctx, user)
	if errors.Is(err, entity.ErrEmailAlreadyExists) {
		return nil, entity.ErrEmailAlreadyExists
	}
	if err != nil {
		return nil, entity.ErrInternal
	}
	return created, nil
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
	now := u.clock.Now()
	access, expiresIn, err := u.tokenService.GenerateAccessToken(user, now)
	if err != nil {
		return nil, entity.ErrInternal
	}
	plain, hash, err := u.tokenService.GenerateRefreshToken()
	if err != nil {
		return nil, entity.ErrInternal
	}
	csrf, err := u.tokenService.GenerateCSRFToken()
	if err != nil {
		return nil, entity.ErrInternal
	}
	expiresAt := now.Add(u.tokenService.RefreshTokenTTL())
	refresh := &entity.RefreshToken{ID: uuid.New(), UserID: user.ID, TokenHash: hash, FamilyID: u.tokenService.GenerateFamilyID(), ExpiresAt: expiresAt, CreatedAt: now}
	if err := u.refreshTokenRepository.Create(ctx, refresh); err != nil {
		return nil, entity.ErrInternal
	}
	return &LoginResult{AccessToken: access, ExpiresIn: expiresIn, RefreshToken: plain, RefreshTokenExpiresAt: expiresAt, CSRFToken: csrf, User: user}, nil
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
	if current.IsRevoked() || current.IsExpired(now) {
		return nil, entity.ErrInvalidRefreshToken
	}
	if current.IsUsed() {
		if err := u.authTransactionManager.RevokeFamilyAndIncrementTokenVersion(ctx, current.FamilyID, current.UserID, now); err != nil {
			return nil, entity.ErrInternal
		}
		return nil, entity.ErrRefreshTokenReused
	}
	user, err := u.userRepository.FindByID(ctx, current.UserID)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			return nil, entity.ErrInvalidRefreshToken
		}
		return nil, entity.ErrInternal
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
		ExpiresAt: expiresAt, CreatedAt: now,
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

func (u *userUsecase) ValidateTokenVersion(ctx context.Context, userID uuid.UUID, tokenVersion uint64) (*entity.User, error) {
	user, err := u.userRepository.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.Status != entity.UserStatusActive || user.TokenVersion != tokenVersion {
		return nil, entity.ErrInvalidCredentials
	}
	return user, nil
}
