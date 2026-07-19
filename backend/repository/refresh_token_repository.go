package repository

import (
	"context"
	"errors"
	"time"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IRefreshTokenRepository interface {
	Create(ctx context.Context, refreshToken *entity.RefreshToken) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error)
	Rotate(ctx context.Context, currentID uuid.UUID, next *entity.RefreshToken, usedAt time.Time) error
	RevokeFamilyAndIncrementTokenVersion(ctx context.Context, familyID, userID uuid.UUID, revokedAt time.Time) error
}

type refreshTokenRepository struct{ db *gorm.DB }

func NewRefreshTokenRepository(db *gorm.DB) IRefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) Create(ctx context.Context, token *entity.RefreshToken) error {
	if token == nil || token.UserID == uuid.Nil || token.FamilyID == uuid.Nil || token.TokenHash == "" || !token.ExpiresAt.After(token.CreatedAt) {
		return entity.ErrInvalidInput
	}
	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		return entity.ErrInternal
	}
	return nil
}

func (r *refreshTokenRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error) {
	if tokenHash == "" {
		return nil, entity.ErrInvalidRefreshToken
	}
	var token entity.RefreshToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, entity.ErrInvalidRefreshToken
	}
	if err != nil {
		return nil, entity.ErrInternal
	}
	return &token, nil
}

func (r *refreshTokenRepository) Rotate(ctx context.Context, currentID uuid.UUID, next *entity.RefreshToken, usedAt time.Time) error {
	if currentID == uuid.Nil || next == nil {
		return entity.ErrInvalidInput
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&entity.RefreshToken{}).
			Where("id = ? AND used_at IS NULL AND revoked_at IS NULL", currentID).
			Updates(map[string]any{"used_at": usedAt, "replaced_by_id": next.ID})
		if result.Error != nil {
			return entity.ErrInternal
		}
		if result.RowsAffected != 1 {
			return entity.ErrRefreshTokenReused
		}
		if err := tx.Create(next).Error; err != nil {
			return entity.ErrInternal
		}
		return nil
	})
}

func (r *refreshTokenRepository) RevokeFamilyAndIncrementTokenVersion(ctx context.Context, familyID, userID uuid.UUID, revokedAt time.Time) error {
	if familyID == uuid.Nil || userID == uuid.Nil {
		return entity.ErrInvalidInput
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.RefreshToken{}).
			Where("family_id = ? AND revoked_at IS NULL", familyID).
			Update("revoked_at", revokedAt).Error; err != nil {
			return entity.ErrInternal
		}
		result := tx.Model(&entity.User{}).Where("id = ?", userID).
			UpdateColumn("token_version", gorm.Expr("token_version + 1"))
		if result.Error != nil || result.RowsAffected != 1 {
			return entity.ErrInternal
		}
		return nil
	})
}
