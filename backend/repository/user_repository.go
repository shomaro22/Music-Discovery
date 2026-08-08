package repository

import (
	"context"
	"errors"
	"time"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type IUserRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	Create(ctx context.Context, user *entity.User) (*entity.User, error)
	SoftDelete(ctx context.Context, tx ITransaction, userID uuid.UUID, now time.Time) error
	IncrementTokenVersion(ctx context.Context, tx ITransaction, userID uuid.UUID, now time.Time) error
	UpdatePassword(ctx context.Context, tx ITransaction, userID uuid.UUID, passwordHash string, now time.Time) error
	UpdateProfile(ctx context.Context, userID uuid.UUID, name, email string, now time.Time) (*entity.User, error)
}

type userRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) IUserRepository { return &userRepository{db: db} }

func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	var user entity.User
	result := r.db.WithContext(ctx).First(&user, "id = ?", id)
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
	if err := assertUserForCreate(user); err != nil {
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
func (r *userRepository) SoftDelete(
	ctx context.Context,
	tx ITransaction,
	userID uuid.UUID,
	now time.Time,
) error {
	db, ok := transactionDB(tx)
	if !ok || userID == uuid.Nil {
		return entity.ErrInvalidInput
	}

	result := db.WithContext(ctx).
		Unscoped().
		Model(&entity.User{}).
		Where("id = ? AND deleted_at IS NULL", userID).
		Updates(struct {
			DeletedAt time.Time
			UpdatedAt time.Time
		}{
			DeletedAt: now,
			UpdatedAt: now,
		})

	if result.Error != nil {
		return entity.ErrInternal
	}
	if result.RowsAffected == 0 {
		var count int64
		checkResult := db.WithContext(ctx).
			Unscoped().
			Model(&entity.User{}).
			Where("id = ?", userID).
			Count(&count)
		if checkResult.Error != nil {
			return entity.ErrInternal
		}
		if count == 0 {
			return entity.ErrUserNotFound
		}
		return entity.ErrUserAlreadyWithdrawn
	}
	return nil
}
func (r *userRepository) IncrementTokenVersion(ctx context.Context, tx ITransaction, userID uuid.UUID, now time.Time) error {
	db, ok := transactionDB(tx)
	if !ok || userID == uuid.Nil {
		return entity.ErrInvalidInput
	}
	result := db.WithContext(ctx).Exec("UPDATE users SET token_version = token_version + 1, updated_at = ? WHERE id = ?", now, userID)
	if result.Error != nil || result.RowsAffected != 1 {
		return entity.ErrInternal
	}
	return nil
}

func (r *userRepository) UpdatePassword(ctx context.Context, tx ITransaction, userID uuid.UUID, passwordHash string, now time.Time) error {
	db, ok := transactionDB(tx)
	if !ok || userID == uuid.Nil || passwordHash == "" {
		return entity.ErrInvalidInput
	}
	result := db.WithContext(ctx).Model(&entity.User{}).
		Where("id = ? AND deleted_at IS NULL", userID).
		Updates(map[string]interface{}{"password_hash": passwordHash, "updated_at": now})
	if result.Error != nil {
		return entity.ErrInternal
	}
	if result.RowsAffected != 1 {
		return entity.ErrUserNotFound
	}
	return nil
}

func assertUserForCreate(user *entity.User) error {
	// 形式、必須項目、およびビジネスロジックに関するバリデーションは、ValidatorやUsecaseの責務です。
	if user == nil {
		return entity.ErrInvalidUserForCreate
	}
	return nil
}

func (r *userRepository) UpdateProfile(ctx context.Context, userID uuid.UUID, name, email string, now time.Time) (*entity.User, error) {
	result := r.db.WithContext(ctx).Model(&entity.User{}).Where("id = ? AND deleted_at IS NULL", userID).Updates(map[string]interface{}{
		"name": name, "email": email, "updated_at": now,
	})
	if result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) && pgErr.Code == "23505" {
			return nil, entity.ErrEmailAlreadyExists
		}
		return nil, entity.ErrInternal
	}
	if result.RowsAffected != 1 {
		return nil, entity.ErrUserNotFound
	}
	return r.FindByID(ctx, userID)
}
