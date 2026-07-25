package repository

import (
	"context"
	"errors"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IQuestionRepository interface {
	ListActive(ctx context.Context) ([]entity.Question, error)
	FindChoicesByIDs(ctx context.Context, ids []uuid.UUID) ([]entity.QuestionChoice, error)
	GetLatestAnswers(ctx context.Context, userID uuid.UUID) (*entity.QuestionAnswerSession, error)
	SaveAnswersAndProfile(ctx context.Context, session *entity.QuestionAnswerSession, profile *entity.UserProfile, genre []entity.UserGenreWeight, element []entity.UserElementWeight, atmosphere []entity.UserAtmosphereWeight) (bool, error)
}

type questionRepository struct{ db *gorm.DB }

func NewQuestionRepository(db *gorm.DB) IQuestionRepository { return &questionRepository{db: db} }

func (r *questionRepository) ListActive(ctx context.Context) ([]entity.Question, error) {
	var questions []entity.Question
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Order("sort_order ASC").Preload("Choices", func(db *gorm.DB) *gorm.DB { return db.Where("is_active = ?", true).Order("sort_order ASC") }).Find(&questions).Error
	if err != nil {
		return nil, entity.ErrInternal
	}
	filtered := make([]entity.Question, 0, len(questions))
	for _, question := range questions {
		if len(question.Choices) > 0 {
			filtered = append(filtered, question)
		}
	}
	return filtered, nil
}

func (r *questionRepository) FindChoicesByIDs(ctx context.Context, ids []uuid.UUID) ([]entity.QuestionChoice, error) {
	if len(ids) == 0 {
		return []entity.QuestionChoice{}, nil
	}
	var choices []entity.QuestionChoice
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&choices).Error; err != nil {
		return nil, entity.ErrInternal
	}
	return choices, nil
}

func (r *questionRepository) GetLatestAnswers(ctx context.Context, userID uuid.UUID) (*entity.QuestionAnswerSession, error) {
	var session entity.QuestionAnswerSession
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("answered_at DESC, created_at DESC").Preload("Answers.Choices").First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, entity.ErrInternal
	}
	return &session, nil
}

func (r *questionRepository) SaveAnswersAndProfile(ctx context.Context, session *entity.QuestionAnswerSession, profile *entity.UserProfile, genre []entity.UserGenreWeight, element []entity.UserElementWeight, atmosphere []entity.UserAtmosphereWeight) (bool, error) {
	if session == nil || profile == nil {
		return false, entity.ErrInvalidInput
	}
	initialized := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(session).Error; err != nil {
			return entity.ErrAnswerSaveFailed
		}
		var existing entity.UserProfile
		result := tx.Where("user_id = ?", profile.UserID).First(&existing)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			initialized = true
			if err := tx.Create(profile).Error; err != nil {
				return entity.ErrProfileCreateFailed
			}
		} else if result.Error != nil {
			return entity.ErrProfileCreateFailed
		} else {
			profile.ID = existing.ID
			profile.CreatedAt = existing.CreatedAt
			if err := tx.Model(&existing).Updates(map[string]interface{}{"discovery_level": profile.DiscoveryLevel, "updated_at": profile.UpdatedAt}).Error; err != nil {
				return entity.ErrProfileCreateFailed
			}
		}
		for i := range genre {
			genre[i].ProfileID = profile.ID
		}
		for i := range element {
			element[i].ProfileID = profile.ID
		}
		for i := range atmosphere {
			atmosphere[i].ProfileID = profile.ID
		}
		if err := upsertWeights(tx, genre); err != nil {
			return entity.ErrProfileCreateFailed
		}
		if err := upsertWeights(tx, element); err != nil {
			return entity.ErrProfileCreateFailed
		}
		if err := upsertWeights(tx, atmosphere); err != nil {
			return entity.ErrProfileCreateFailed
		}
		return nil
	})
	return initialized, err
}

func upsertWeights[T entity.UserGenreWeight | entity.UserElementWeight | entity.UserAtmosphereWeight](tx *gorm.DB, values []T) error {
	if len(values) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "profile_id"}, {Name: "choice_id"}}, DoUpdates: clause.AssignmentColumns([]string{"weight", "updated_at"})}).Create(&values).Error
}
