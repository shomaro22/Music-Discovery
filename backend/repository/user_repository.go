package repository

import (
	"context"
	"errors"
	"strings"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type IUserRepository interface {
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	Create(ctx context.Context, user *entity.User) (*entity.User, error)
}

type userRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) IUserRepository { return &userRepository{db: db} }

func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	if id == uuid.Nil {
		return nil, entity.ErrInvalidInput
	}
	var user entity.User
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, entity.ErrUserNotFound
	}
	if result.Error != nil {
		return nil, entity.ErrInternal
	}
	return &user, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	result := r.db.WithContext(ctx).Where("email = ?", email).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, entity.ErrUserNotFound
	}
	if result.Error != nil {
		return nil, entity.ErrInternal
	}
	return &user, nil
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) (*entity.User, error) {
	if err := validateUserForCreate(user); err != nil {
		return nil, err
	}

	result := r.db.WithContext(ctx).Create(user)
	if result.Error == nil {
		return user, nil
	}

	var pgErr *pgconn.PgError
	if errors.As(result.Error, &pgErr) && pgErr.Code == "23505" {
		return nil, entity.ErrEmailAlreadyExists
	}
	return nil, entity.ErrInternal
}

func validateUserForCreate(user *entity.User) error {
	if user == nil {
		return entity.ErrInvalidUserForCreate
	}
	if strings.TrimSpace(user.Email) == "" || strings.TrimSpace(user.PasswordHash) == "" {
		return entity.ErrInvalidUserForCreate
	}
	if user.Role == "" || user.Status == "" {
		return entity.ErrInvalidUserForCreate
	}
	return nil
}
