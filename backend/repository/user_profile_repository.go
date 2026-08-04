package repository

import (
	"context"
	"errors"
	"time"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IUserProfileRepository interface {
	FindByUserID(ctx context.Context, tx ITransaction, userID uuid.UUID) (*entity.UserProfile, error)
	Create(ctx context.Context, tx ITransaction, profile *entity.UserProfile) error
	UpdateDiscoveryLevel(ctx context.Context, tx ITransaction, profileID uuid.UUID, discoveryLevel string, updatedAt time.Time) error
	ResetQuestionWeights(ctx context.Context, tx ITransaction, profileID uuid.UUID, updatedAt time.Time) error
	UpsertGenreWeights(ctx context.Context, tx ITransaction, values []entity.UserGenreWeight) error
	UpsertElementWeights(ctx context.Context, tx ITransaction, values []entity.UserElementWeight) error
	UpsertAtmosphereWeights(ctx context.Context, tx ITransaction, values []entity.UserAtmosphereWeight) error
}

type IUserProfileTimezoneRepository interface {
	FindTimezoneByUserID(ctx context.Context, userID uuid.UUID) (string, error)
}

type IUserProfileStore interface {
	IUserProfileRepository
	IUserProfileTimezoneRepository
}

type userProfileRepository struct{ db *gorm.DB }

func NewUserProfileRepository(db *gorm.DB) IUserProfileStore {
	return &userProfileRepository{db: db}
}

func (r *userProfileRepository) FindByUserID(ctx context.Context, tx ITransaction, userID uuid.UUID) (*entity.UserProfile, error) {
	db, ok := transactionDB(tx)
	if !ok {
		return nil, entity.ErrInvalidInput
	}
	var profile entity.UserProfile
	err := db.WithContext(ctx).Where("user_id = ?", userID).
		Preload("GenreWeights").Preload("ElementWeights").Preload("AtmosphereWeights").
		First(&profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, entity.ErrProfileCreateFailed
	}
	return &profile, nil
}

func (r *userProfileRepository) FindTimezoneByUserID(ctx context.Context, userID uuid.UUID) (string, error) {
	var profile struct {
		Timezone string `gorm:"column:timezone"`
	}
	err := r.db.WithContext(ctx).
		Model(&entity.UserProfile{}).
		Select("timezone").
		Where("user_id = ?", userID).
		Take(&profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", entity.ErrNotFound
	}
	if err != nil {
		return "", entity.ErrProfileCreateFailed
	}
	return profile.Timezone, nil
}

func (r *userProfileRepository) Create(ctx context.Context, tx ITransaction, profile *entity.UserProfile) error {
	db, ok := transactionDB(tx)
	if !ok || profile == nil {
		return entity.ErrInvalidInput
	}
	if err := db.WithContext(ctx).Create(profile).Error; err != nil {
		return entity.ErrProfileCreateFailed
	}
	return nil
}

func (r *userProfileRepository) UpdateDiscoveryLevel(ctx context.Context, tx ITransaction, profileID uuid.UUID, discoveryLevel string, updatedAt time.Time) error {
	db, ok := transactionDB(tx)
	if !ok {
		return entity.ErrInvalidInput
	}
	if err := db.WithContext(ctx).Model(&entity.UserProfile{}).Where("id = ?", profileID).
		Updates(struct {
			DiscoveryLevel string    `gorm:"column:discovery_level"`
			UpdatedAt      time.Time `gorm:"column:updated_at"`
		}{DiscoveryLevel: discoveryLevel, UpdatedAt: updatedAt}).Error; err != nil {
		return entity.ErrProfileCreateFailed
	}
	return nil
}

func (r *userProfileRepository) ResetQuestionWeights(ctx context.Context, tx ITransaction, profileID uuid.UUID, updatedAt time.Time) error {
	db, ok := transactionDB(tx)
	if !ok {
		return entity.ErrInvalidInput
	}
	models := []interface{}{&entity.UserGenreWeight{}, &entity.UserElementWeight{}, &entity.UserAtmosphereWeight{}}
	for _, model := range models {
		if err := db.WithContext(ctx).Model(model).Where("profile_id = ?", profileID).Updates(map[string]interface{}{
			"question_weight": 0,
			"weight":          gorm.Expr("feedback_weight"),
			"updated_at":      updatedAt,
		}).Error; err != nil {
			return entity.ErrProfileCreateFailed
		}
	}
	return nil
}

func (r *userProfileRepository) UpsertGenreWeights(ctx context.Context, tx ITransaction, values []entity.UserGenreWeight) error {
	return upsertQuestionWeights(ctx, tx, values)
}
func (r *userProfileRepository) UpsertElementWeights(ctx context.Context, tx ITransaction, values []entity.UserElementWeight) error {
	return upsertQuestionWeights(ctx, tx, values)
}
func (r *userProfileRepository) UpsertAtmosphereWeights(ctx context.Context, tx ITransaction, values []entity.UserAtmosphereWeight) error {
	return upsertQuestionWeights(ctx, tx, values)
}

func upsertQuestionWeights[T entity.UserGenreWeight | entity.UserElementWeight | entity.UserAtmosphereWeight](ctx context.Context, tx ITransaction, values []T) error {
	if len(values) == 0 {
		return nil
	}
	db, ok := transactionDB(tx)
	if !ok {
		return entity.ErrInvalidInput
	}
	conflict := clause.OnConflict{
		Columns: []clause.Column{{Name: "profile_id"}, {Name: "choice_id"}},
		DoUpdates: clause.Set{
			{Column: clause.Column{Name: "question_weight"}, Value: gorm.Expr("excluded.question_weight")},
			{Column: clause.Column{Name: "weight"}, Value: gorm.Expr("excluded.weight")},
			{Column: clause.Column{Name: "updated_at"}, Value: gorm.Expr("excluded.updated_at")},
		},
	}
	if err := db.WithContext(ctx).Clauses(conflict).Create(&values).Error; err != nil {
		return entity.ErrProfileCreateFailed
	}
	return nil
}
