package repository

import (
	"context"
	"gorm.io/gorm"
	"music-discovery/backend/entity"
	"time"
)

type logRetentionRepository struct{ db *gorm.DB }

func NewLogRetentionRepository(db *gorm.DB) *logRetentionRepository {
	return &logRetentionRepository{db: db}
}
func (r *logRetentionRepository) DeleteExpiredSystemLogs(ctx context.Context, standardCutoff, sensitiveCutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Where("(category IN ? OR level = ?) AND created_at < ? OR (category NOT IN ? AND level <> ? AND created_at < ?)", []string{"security"}, "error", sensitiveCutoff, []string{"security"}, "error", standardCutoff).Delete(&entity.SystemLog{})
	return result.RowsAffected, result.Error
}
func (r *logRetentionRepository) DeleteExpiredAuditLogs(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Where("created_at < ?", cutoff).Delete(&entity.AuditLog{})
	return result.RowsAffected, result.Error
}
