package repository

import (
	"context"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IQuestionRepository interface {
	ListActive(ctx context.Context) ([]entity.Question, error)
	FindChoicesByIDs(ctx context.Context, ids []uuid.UUID) ([]entity.QuestionChoice, error)
}

type questionRepository struct{ db *gorm.DB }

func NewQuestionRepository(db *gorm.DB) IQuestionRepository { return &questionRepository{db: db} }

func (r *questionRepository) ListActive(ctx context.Context) ([]entity.Question, error) {
	var questions []entity.Question
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("sort_order ASC").
		Preload("Choices", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_active = ?", true).Order("sort_order ASC")
		}).
		Find(&questions).Error
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
