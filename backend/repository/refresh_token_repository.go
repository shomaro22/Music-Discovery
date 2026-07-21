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
	Create(context.Context, *entity.RefreshToken) error
	FindByTokenHash(context.Context, string) (*entity.RefreshToken, error)
	Rotate(context.Context, uuid.UUID, *entity.RefreshToken, time.Time) error
	RevokeFamily(context.Context, uuid.UUID, time.Time) error
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
		return nil, entity.ErrInvalidRefreshToken
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
func (r *refreshTokenRepository) Rotate(ctx context.Context, currentID uuid.UUID, next *entity.RefreshToken, now time.Time) error {
	if currentID == uuid.Nil || next == nil {
		return entity.ErrInvalidInput
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current entity.RefreshToken
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, "id = ?", currentID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.ErrInvalidRefreshToken
		}
		if err != nil {
			return entity.ErrInternal
		}
		if current.IsRevoked() {
			return entity.ErrRefreshTokenReused
		}
		if current.IsExpired(now) {
			return entity.ErrInvalidRefreshToken
		}
		revokedAt := now
		updates := entity.RefreshToken{
			RevokedAt:    &revokedAt,
			ReplacedByID: &next.ID,
		}
		if err := tx.Model(&current).Select("revoked_at", "replaced_by_id").Updates(updates).Error; err != nil {
			return entity.ErrInternal
		}
		if err := tx.Create(next).Error; err != nil {
			return entity.ErrInternal
		}
		return nil
	})
}

func (r *refreshTokenRepository) RevokeFamily(ctx context.Context, familyID uuid.UUID, now time.Time) error {
	if familyID == uuid.Nil {
		return entity.ErrInvalidInput
	}
	if err := r.db.WithContext(ctx).Model(&entity.RefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", now).Error; err != nil {
		return entity.ErrInternal
	}
	return nil
}
