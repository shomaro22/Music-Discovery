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
	ITransactionManager

	CreateUserAndRefreshToken(ctx context.Context, user *entity.User, token *entity.RefreshToken) error
	RevokeFamilyAndIncrementTokenVersion(ctx context.Context, familyID, userID uuid.UUID, now time.Time) error
}

type IApplicationTransactionManager interface {
	IAuthTransactionManager
	IRoleTransactionManager
}

type authTransactionManager struct{ db *gorm.DB }

var _ IApplicationTransactionManager = (*authTransactionManager)(nil)

func NewAuthTransactionManager(db *gorm.DB) IApplicationTransactionManager {
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
		result := tx.Model(&entity.RefreshToken{}).
			Where("family_id = ? AND revoked_at IS NULL", familyID).
			Update("revoked_at", now)
		if result.Error != nil {
			return entity.ErrInternal
		}
		if result.RowsAffected == 0 {
			return entity.ErrInvalidRefreshToken
		}

		result = tx.Exec(
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

func (m *authTransactionManager) WithinTransaction(ctx context.Context, fn func(tx ITransaction) error) error {
	if m == nil || m.db == nil || fn == nil {
		return entity.ErrInvalidInput
	}
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&gormTransaction{db: tx})
	})
}

type IRoleTransactionManager interface {
	ChangeUserRole(ctx context.Context, userID uuid.UUID, role entity.Role, now time.Time) error
}

func (m *authTransactionManager) ChangeUserRole(ctx context.Context, userID uuid.UUID, role entity.Role, now time.Time) error {
	if userID == uuid.Nil || !role.IsPersistable() {
		return entity.ErrInvalidInput
	}
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec(
			"UPDATE users SET role = ?, token_version = token_version + 1, updated_at = ? WHERE id = ? AND deleted_at IS NULL",
			role, now, userID,
		)
		if result.Error != nil {
			return entity.ErrInternal
		}
		if result.RowsAffected == 0 {
			return entity.ErrUserNotFound
		}
		if err := tx.Model(&entity.RefreshToken{}).Where("user_id = ? AND revoked_at IS NULL", userID).Update("revoked_at", now).Error; err != nil {
			return entity.ErrInternal
		}
		return nil
	})
}
