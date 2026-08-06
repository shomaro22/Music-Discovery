package repository

import (
	"context"
	"errors"
	"time"

	"music-discovery/backend/entity"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type alertStateRepository struct{ db *gorm.DB }

func NewAlertStateRepository(db *gorm.DB) *alertStateRepository { return &alertStateRepository{db: db} }
func (r *alertStateRepository) List(ctx context.Context) ([]entity.AlertStateRecord, error) {
	var values []entity.AlertStateRecord
	err := r.db.WithContext(ctx).Order("code ASC").Find(&values).Error
	return values, err
}
func (r *alertStateRepository) FindByCode(ctx context.Context, code string) (*entity.AlertStateRecord, error) {
	var value entity.AlertStateRecord
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&value).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &value, err
}
func (r *alertStateRepository) UpsertState(ctx context.Context, value *entity.AlertStateRecord) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "code"}}, DoUpdates: clause.AssignmentColumns([]string{"severity", "status", "message", "value", "threshold", "updated_at"})}).Create(value).Error
}
func (r *alertStateRepository) MarkNotified(ctx context.Context, code, status string, notifiedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.AlertStateRecord{}).Where("code = ?", code).Updates(map[string]interface{}{"last_notified_status": status, "last_notified_at": notifiedAt, "updated_at": notifiedAt}).Error
}
