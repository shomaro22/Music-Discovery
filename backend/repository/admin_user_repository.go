package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"music-discovery/backend/entity"
)

func (r *adminRepository) ListUsers(ctx context.Context, q PageQuery) ([]entity.User, int64, error) {
	db := r.db.WithContext(ctx).Model(&entity.User{})
	if q.IncludeDeleted {
		db = db.Unscoped()
	}
	db = addKeyword(db, q.Keyword, "name", "email")
	if q.Role != "" {
		db = db.Where("role = ?", q.Role)
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, repoError(err)
	}
	var items []entity.User
	if err := db.Order(safeOrder(q.Sort, q.Order, map[string]string{"name": "name", "email": "email", "role": "role", "status": "status", "created_at": "created_at", "updated_at": "updated_at", "deleted_at": "deleted_at"}, "created_at")).Offset(offset(q)).Limit(q.Limit).Find(&items).Error; err != nil {
		return nil, 0, repoError(err)
	}
	return items, total, nil
}
func (r *adminRepository) FindUserUnscoped(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	var value entity.User
	if err := r.db.WithContext(ctx).Unscoped().First(&value, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, entity.ErrUserNotFound
		}
		return nil, repoError(err)
	}
	return &value, nil
}

func (r *adminRepository) FindUserMetrics(ctx context.Context, userID uuid.UUID) (*UserMetrics, error) {
	value := &UserMetrics{ProfileStatus: "not_created"}
	if r.db.Migrator().HasTable("user_profiles") {
		var count int64
		if err := r.db.WithContext(ctx).Table("user_profiles").Where("user_id = ?", userID).Count(&count).Error; err != nil {
			return nil, repoError(err)
		}
		if count > 0 {
			value.ProfileStatus = "created"
		}
	}
	if r.db.Migrator().HasTable("recommendations") {
		if err := r.db.WithContext(ctx).Table("recommendations").Where("user_id = ?", userID).Count(&value.RecommendationCount).Error; err != nil {
			return nil, repoError(err)
		}
	}
	if r.db.Migrator().HasTable("feedbacks") {
		if err := r.db.WithContext(ctx).Table("feedbacks").Where("user_id = ?", userID).Count(&value.FeedbackCount).Error; err != nil {
			return nil, repoError(err)
		}
	}
	if r.db.Migrator().HasTable("login_histories") {
		var row struct{ LoggedInAt time.Time }
		result := r.db.WithContext(ctx).Table("login_histories").Select("logged_in_at").Where("user_id = ?", userID).Order("logged_in_at DESC").Limit(1).Scan(&row)
		if result.Error != nil {
			return nil, repoError(result.Error)
		}
		if !row.LoggedInAt.IsZero() {
			value.LastLoginAt = &row.LoggedInAt
		}
	}
	return value, nil
}

func activeSystemAdminCount(tx *gorm.DB) (int64, error) {
	var count int64
	err := tx.Model(&entity.User{}).Where("role = ? AND status = ? AND deleted_at IS NULL", entity.RoleSystemAdmin, entity.UserStatusActive).Count(&count).Error
	return count, err
}
func (r *adminRepository) SetUserStatusWithAudit(ctx context.Context, actorID, userID uuid.UUID, status entity.UserStatus, reason string, now time.Time, a AuditContext) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user entity.User
		if err := tx.Unscoped().First(&user, "id = ?", userID).Error; err != nil {
			return repoError(err)
		}
		if user.DeletedAt.Valid {
			return entity.ErrUserNotFound
		}
		if user.Status == status {
			if status == entity.UserStatusActive {
				return entity.ErrAlreadyActive
			}
			return entity.ErrAlreadySuspended
		}
		if status == entity.UserStatusSuspended && user.Role == entity.RoleSystemAdmin {
			count, err := activeSystemAdminCount(tx)
			if err != nil {
				return repoError(err)
			}
			if count <= 1 {
				return entity.ErrLastSystemAdmin
			}
		}
		updates := map[string]interface{}{"status": status, "token_version": gorm.Expr("token_version + 1"), "updated_at": now}
		if status == entity.UserStatusSuspended {
			updates["suspended_at"] = now
			updates["suspended_reason"] = reason
			updates["suspended_by"] = actorID
		} else {
			updates["suspended_at"] = nil
			updates["suspended_reason"] = ""
			updates["suspended_by"] = nil
		}
		if err := tx.Model(&entity.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
			return repoError(err)
		}
		if err := tx.Model(&entity.RefreshToken{}).Where("user_id = ? AND revoked_at IS NULL", userID).Update("revoked_at", now).Error; err != nil {
			return repoError(err)
		}
		a.TargetID = userID.String()
		a.Before = map[string]interface{}{"status": user.Status}
		a.After = map[string]interface{}{"status": status}
		return createAudit(tx, a)
	})
}
func (r *adminRepository) ChangeUserRoleWithAudit(ctx context.Context, userID uuid.UUID, role entity.Role, now time.Time, a AuditContext) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user entity.User
		if err := tx.Unscoped().First(&user, "id = ?", userID).Error; err != nil {
			return repoError(err)
		}
		if user.DeletedAt.Valid {
			return entity.ErrUserNotFound
		}
		if user.Role == role {
			return entity.ErrConflict
		}
		if user.Role == entity.RoleSystemAdmin && role != entity.RoleSystemAdmin && user.Status == entity.UserStatusActive {
			count, err := activeSystemAdminCount(tx)
			if err != nil {
				return repoError(err)
			}
			if count <= 1 {
				return entity.ErrLastSystemAdmin
			}
		}
		if err := tx.Model(&entity.User{}).Where("id = ?", userID).Updates(map[string]interface{}{"role": role, "token_version": gorm.Expr("token_version + 1"), "updated_at": now}).Error; err != nil {
			return repoError(err)
		}
		if err := tx.Model(&entity.RefreshToken{}).Where("user_id = ? AND revoked_at IS NULL", userID).Update("revoked_at", now).Error; err != nil {
			return repoError(err)
		}
		a.TargetID = userID.String()
		a.Before = map[string]interface{}{"role": user.Role}
		a.After = map[string]interface{}{"role": role}
		return createAudit(tx, a)
	})
}
