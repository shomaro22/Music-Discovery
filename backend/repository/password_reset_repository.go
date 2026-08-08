package repository

import (
	"context"
	"errors"
	"time"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IPasswordResetRepository interface {
	Create(ctx context.Context, token *entity.PasswordResetToken) error
	FindByHashForUpdate(ctx context.Context, tx ITransaction, tokenHash string) (*entity.PasswordResetToken, error)
	MarkUsed(ctx context.Context, tx ITransaction, tokenID uuid.UUID, now time.Time) error
}

type passwordResetRepository struct{ db *gorm.DB }

func NewPasswordResetRepository(db *gorm.DB) IPasswordResetRepository {
	return &passwordResetRepository{db: db}
}

func (r *passwordResetRepository) Create(ctx context.Context, token *entity.PasswordResetToken) error {
	if token == nil || token.UserID == uuid.Nil || token.TokenHash == "" {
		return entity.ErrInvalidInput
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND used_at IS NULL", token.UserID).Delete(&entity.PasswordResetToken{}).Error; err != nil {
			return entity.ErrInternal
		}
		if err := tx.Create(token).Error; err != nil {
			return entity.ErrInternal
		}
		return nil
	})
}

func (r *passwordResetRepository) FindByHashForUpdate(ctx context.Context, tx ITransaction, tokenHash string) (*entity.PasswordResetToken, error) {
	db, ok := transactionDB(tx)
	if !ok || tokenHash == "" {
		return nil, entity.ErrInvalidInput
	}
	var token entity.PasswordResetToken
	result := db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("token_hash = ?", tokenHash).First(&token)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, entity.ErrInvalidPasswordResetToken
	}
	if result.Error != nil {
		return nil, entity.ErrInternal
	}
	return &token, nil
}

func (r *passwordResetRepository) MarkUsed(ctx context.Context, tx ITransaction, tokenID uuid.UUID, now time.Time) error {
	db, ok := transactionDB(tx)
	if !ok || tokenID == uuid.Nil {
		return entity.ErrInvalidInput
	}
	result := db.WithContext(ctx).Model(&entity.PasswordResetToken{}).
		Where("id = ? AND used_at IS NULL", tokenID).
		Updates(map[string]interface{}{"used_at": now})
	if result.Error != nil {
		return entity.ErrInternal
	}
	if result.RowsAffected != 1 {
		return entity.ErrInvalidPasswordResetToken
	}
	return nil
}
