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

type IAuthTransactionManager interface {
	CreateUserAndRefreshToken(context.Context, *entity.User, *entity.RefreshToken) error
	RevokeFamilyAndIncrementTokenVersion(context.Context, uuid.UUID, uuid.UUID, time.Time) error
}

type authTransactionManager struct{ db *gorm.DB }

func NewAuthTransactionManager(db *gorm.DB) IAuthTransactionManager {
	return &authTransactionManager{db: db}
}

func (m *authTransactionManager) CreateUserAndRefreshToken(ctx context.Context, user *entity.User, token *entity.RefreshToken) error {
	if user == nil || token == nil || user.ID == uuid.Nil || token.UserID != user.ID {
		return entity.ErrInvalidInput
	}
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return entity.ErrEmailAlreadyExists
			}
			return entity.ErrInternal
		}
		if err := tx.Create(token).Error; err != nil {
			return entity.ErrInternal
		}
		return nil
	})
}

func (m *authTransactionManager) RevokeFamilyAndIncrementTokenVersion(ctx context.Context, familyID, userID uuid.UUID, now time.Time) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.RefreshToken{}).
			Where("family_id = ? AND revoked_at IS NULL", familyID).
			Update("revoked_at", now).Error; err != nil {
			return entity.ErrInternal
		}

		result := tx.Exec(
			"UPDATE users SET token_version = token_version + 1, updated_at = ? WHERE id = ? AND deleted_at IS NULL",
			now,
			userID,
		)
		if result.Error != nil || result.RowsAffected != 1 {
			return entity.ErrInternal
		}
		return nil
	})
}
