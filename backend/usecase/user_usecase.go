package usecase

import (
	"context"
	"errors"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/repository"

	"golang.org/x/crypto/bcrypt"
)

type SignUpInput struct {
	Name     string
	Email    string
	Password string
}

type IPasswordHasher interface {
	Hash(password string) (string, error)
}

type bcryptPasswordHasher struct{}

func NewPasswordHasher() IPasswordHasher {
	return &bcryptPasswordHasher{}
}

func (h *bcryptPasswordHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", entity.ErrInternal
	}
	return string(hash), nil
}

type IClock interface {
	Now() time.Time
}

type systemClock struct{}

func NewClock() IClock                { return &systemClock{} }
func (c *systemClock) Now() time.Time { return time.Now().UTC() }

type IUserUsecase interface {
	SignUp(ctx context.Context, input SignUpInput) (*entity.User, error)
}

type userUsecase struct {
	userRepository repository.IUserRepository
	passwordHasher IPasswordHasher
	clock          IClock
}

func NewUserUsecase(
	userRepository repository.IUserRepository,
	passwordHasher IPasswordHasher,
	clock IClock,
) IUserUsecase {
	return &userUsecase{
		userRepository: userRepository,
		passwordHasher: passwordHasher,
		clock:          clock,
	}
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
		// Continue registration.
	default:
		return nil, entity.ErrInternal
	}

	passwordHash, err := u.passwordHasher.Hash(input.Password)
	if err != nil {
		return nil, entity.ErrInternal
	}

	now := u.clock.Now()
	user := &entity.User{
		Name:         input.Name,
		Email:        input.Email,
		PasswordHash: passwordHash,
		Role:         entity.UserRoleUser,
		Status:       entity.UserStatusActive,
		TokenVersion: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	createdUser, err := u.userRepository.Create(ctx, user)
	if errors.Is(err, entity.ErrEmailAlreadyExists) {
		return nil, entity.ErrEmailAlreadyExists
	}
	if err != nil {
		return nil, entity.ErrInternal
	}
	return createdUser, nil
}
