package repository

import (
	"context"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IRefreshTokenRepository interface {
	Create(ctx context.Context, refreshToken *entity.RefreshToken) error
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
