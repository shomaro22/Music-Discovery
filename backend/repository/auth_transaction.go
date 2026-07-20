package repository

import (
	"context"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"music-discovery/backend/entity"
	"time"
)

type IAuthTransactionManager interface {
	RevokeFamilyAndIncrementTokenVersion(context.Context, uuid.UUID, uuid.UUID, time.Time) error
}
type authTransactionManager struct{ db *gorm.DB }

func NewAuthTransactionManager(db *gorm.DB) IAuthTransactionManager {
	return &authTransactionManager{db: db}
}
func (m *authTransactionManager) RevokeFamilyAndIncrementTokenVersion(ctx context.Context, familyID, userID uuid.UUID, now time.Time) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		revokeResult := tx.Model(&entity.RefreshToken{}).Where("family_id = ? AND revoked_at IS NULL", familyID).Update("revoked_at", now)
		if revokeResult.Error != nil {
			return entity.ErrInternal
		}
		if revokeResult.RowsAffected == 0 {
			return entity.ErrInvalidRefreshToken
		}

		result := tx.Model(&entity.User{}).Where("id = ?", userID).Updates(map[string]any{"token_version": gorm.Expr("token_version + 1"), "updated_at": now})
		if result.Error != nil || result.RowsAffected != 1 {
			return entity.ErrInternal
		}
		return nil
	})
}
