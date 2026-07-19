package repository

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"music-discovery/backend/entity"
	"time"
)

type IRefreshTokenRepository interface {
	Create(ctx context.Context, refreshToken *entity.RefreshToken) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error)
	Rotate(ctx context.Context, oldTokenHash string, newToken *entity.RefreshToken, now time.Time) error
	RevokeFamilyAndIncrementTokenVersion(ctx context.Context, userID, familyID uuid.UUID, now time.Time) error
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
func (r *refreshTokenRepository) FindByTokenHash(ctx context.Context, hash string) (*entity.RefreshToken, error) {
	if hash == "" {
		return nil, entity.ErrInvalidInput
	}
	var token entity.RefreshToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, entity.ErrInvalidRefreshToken
	}
	if err != nil {
		return nil, entity.ErrInternal
	}
	return &token, nil
}
func (r *refreshTokenRepository) Rotate(ctx context.Context, oldHash string, newToken *entity.RefreshToken, now time.Time) error {
	if oldHash == "" || newToken == nil {
		return entity.ErrInvalidInput
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var old entity.RefreshToken
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("token_hash = ?", oldHash).First(&old).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.ErrInvalidRefreshToken
		}
		if err != nil {
			return entity.ErrInternal
		}
		if old.IsRevoked() {
			return entity.ErrRefreshTokenRevoked
		}
		if old.IsUsed() {
			return entity.ErrRefreshTokenUsed
		}
		if old.IsExpired(now) {
			return entity.ErrRefreshTokenExpired
		}
		if err := tx.Create(newToken).Error; err != nil {
			return entity.ErrInternal
		}
		result := tx.Model(&entity.RefreshToken{}).Where("id = ? AND used_at IS NULL AND revoked_at IS NULL", old.ID).Updates(map[string]any{"used_at": now, "replaced_by_id": newToken.ID})
		if result.Error != nil || result.RowsAffected != 1 {
			return entity.ErrRefreshTokenUsed
		}
		return nil
	})
}
func (r *refreshTokenRepository) RevokeFamilyAndIncrementTokenVersion(ctx context.Context, userID, familyID uuid.UUID, now time.Time) error {
	if userID == uuid.Nil || familyID == uuid.Nil {
		return entity.ErrInvalidInput
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.RefreshToken{}).Where("family_id = ? AND revoked_at IS NULL", familyID).Update("revoked_at", now).Error; err != nil {
			return entity.ErrInternal
		}
		var user entity.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", userID).First(&user).Error; err != nil {
			return entity.ErrInternal
		}
		result := tx.Model(&entity.User{}).Where("id = ?", userID).Updates(map[string]any{"token_version": gorm.Expr("token_version + 1"), "updated_at": now})
		if result.Error != nil || result.RowsAffected != 1 {
			return entity.ErrInternal
		}
		return nil
	})
}
