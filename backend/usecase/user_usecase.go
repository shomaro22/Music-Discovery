package usecase

import (
	"context"
	"errors"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/repository"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type SignUpInput struct{ Name, Email, Password string }
type LoginInput struct{ Email, Password string }
type RefreshResult struct {
	AccessToken           string
	ExpiresIn             int64
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
	CSRFToken             string
}

type LoginResult struct {
	AccessToken           string
	ExpiresIn             int64
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
	CSRFToken             string
	User                  *entity.User
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
	Refresh(ctx context.Context, plainRefreshToken string) (*RefreshResult, error)
}

type userUsecase struct {
	userRepository         repository.IUserRepository
	refreshTokenRepository repository.IRefreshTokenRepository
	passwordHasher         IPasswordHasher
	tokenService           ITokenService
	clock                  IClock
}

func NewUserUsecase(userRepository repository.IUserRepository, passwordHasher IPasswordHasher, clock IClock, optional ...any) IUserUsecase {
	u := &userUsecase{userRepository: userRepository, passwordHasher: passwordHasher, clock: clock}
	for _, dep := range optional {
		switch v := dep.(type) {
		case repository.IRefreshTokenRepository:
			u.refreshTokenRepository = v
		case ITokenService:
			u.tokenService = v
		}
	}
	return u
}

func (u *userUsecase) SignUp(ctx context.Context, input SignUpInput) (*entity.User, error) {
	if input.Name == "" || input.Email == "" || input.Password == "" {
		return nil, entity.ErrInvalidInput
	}
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
	if input.Email == "" || input.Password == "" {
		return nil, entity.ErrInvalidInput
	}
	if u.tokenService == nil || u.refreshTokenRepository == nil {
		return nil, entity.ErrInternal
	}
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
	refresh := &entity.RefreshToken{UserID: user.ID, TokenHash: hash, FamilyID: u.tokenService.GenerateFamilyID(), ExpiresAt: expiresAt, CreatedAt: now}
	if err := u.refreshTokenRepository.Create(ctx, refresh); err != nil {
		return nil, entity.ErrInternal
	}
	return &LoginResult{AccessToken: access, ExpiresIn: expiresIn, RefreshToken: plain, RefreshTokenExpiresAt: expiresAt, CSRFToken: csrf, User: user}, nil
}

func (u *userUsecase) Refresh(ctx context.Context, plain string) (*RefreshResult, error) {
	if plain == "" || u.tokenService == nil || u.refreshTokenRepository == nil {
		return nil, entity.ErrInvalidRefreshToken
	}
	hash, err := u.tokenService.HashRefreshToken(plain)
	if err != nil {
		return nil, entity.ErrInvalidRefreshToken
	}
	old, err := u.refreshTokenRepository.FindByTokenHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	now := u.clock.Now()
	user, err := u.userRepository.FindByID(ctx, old.UserID)
	if err != nil {
		return nil, entity.ErrInvalidRefreshToken
	}
	if user.Status == entity.UserStatusSuspended {
		return nil, entity.ErrUserSuspended
	}
	if user.Status != entity.UserStatusActive {
		return nil, entity.ErrInvalidRefreshToken
	}
	if old.IsUsed() {
		return nil, u.handleRefreshTokenReuse(ctx, old, now)
	}
	if old.IsRevoked() {
		return nil, entity.ErrRefreshTokenRevoked
	}
	if old.IsExpired(now) {
		return nil, entity.ErrRefreshTokenExpired
	}
	newPlain, newHash, err := u.tokenService.GenerateRefreshToken()
	if err != nil {
		return nil, entity.ErrInternal
	}
	expiresAt := now.Add(u.tokenService.RefreshTokenTTL())
	next := &entity.RefreshToken{UserID: old.UserID, TokenHash: newHash, FamilyID: old.FamilyID, ExpiresAt: expiresAt, CreatedAt: now}
	if err := u.refreshTokenRepository.Rotate(ctx, hash, next, now); err != nil {
		if errors.Is(err, entity.ErrRefreshTokenUsed) {
			return nil, u.handleRefreshTokenReuse(ctx, old, now)
		}
		return nil, err
	}
	access, expiresIn, err := u.tokenService.GenerateAccessToken(user, now)
	if err != nil {
		return nil, entity.ErrInternal
	}
	csrf, err := u.tokenService.GenerateCSRFToken()
	if err != nil {
		return nil, entity.ErrInternal
	}
	return &RefreshResult{AccessToken: access, ExpiresIn: expiresIn, RefreshToken: newPlain, RefreshTokenExpiresAt: expiresAt, CSRFToken: csrf}, nil
}
func (u *userUsecase) handleRefreshTokenReuse(ctx context.Context, token *entity.RefreshToken, now time.Time) error {
	if token == nil || token.UserID == uuid.Nil || token.FamilyID == uuid.Nil {
		return entity.ErrInternal
	}
	if err := u.refreshTokenRepository.RevokeFamilyAndIncrementTokenVersion(ctx, token.UserID, token.FamilyID, now); err != nil {
		return entity.ErrInternal
	}
	return entity.ErrRefreshTokenReuseDetected
}
