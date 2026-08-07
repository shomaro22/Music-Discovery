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
	ResetPassword(ctx context.Context, tokenHash, passwordHash string, now time.Time) error
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

func (r *passwordResetRepository) ResetPassword(ctx context.Context, tokenHash, passwordHash string, now time.Time) error {
	if tokenHash == "" || passwordHash == "" {
		return entity.ErrInvalidInput
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var token entity.PasswordResetToken
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("token_hash = ?", tokenHash).First(&token)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return entity.ErrInvalidPasswordResetToken
		}
		if result.Error != nil {
			return entity.ErrInternal
		}
		if token.UsedAt != nil || !now.Before(token.ExpiresAt) {
			return entity.ErrInvalidPasswordResetToken
		}

		userResult := tx.Model(&entity.User{}).
			Where("id = ? AND deleted_at IS NULL", token.UserID).
			Updates(map[string]interface{}{"password_hash": passwordHash, "token_version": gorm.Expr("token_version + 1"), "updated_at": now})
		if userResult.Error != nil {
			return entity.ErrInternal
		}
		if userResult.RowsAffected != 1 {
			return entity.ErrInvalidPasswordResetToken
		}
		if err := tx.Model(&entity.RefreshToken{}).
			Where("user_id = ? AND revoked_at IS NULL", token.UserID).
			Updates(map[string]interface{}{"revoked_at": now, "updated_at": now}).Error; err != nil {
			return entity.ErrInternal
		}
		if err := tx.Model(&entity.PasswordResetToken{}).
			Where("id = ? AND used_at IS NULL", token.ID).
			Update("used_at", now).Error; err != nil {
			return entity.ErrInternal
		}
		return nil
	})
}
