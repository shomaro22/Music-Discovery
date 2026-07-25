package usecase

import (
	"context"
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
