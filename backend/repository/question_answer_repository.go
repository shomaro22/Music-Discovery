package repository

import (
	"context"
	"errors"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IQuestionAnswerRepository interface {
	CreateSession(ctx context.Context, tx ITransaction, session *entity.QuestionAnswerSession) error
	GetLatest(ctx context.Context, userID uuid.UUID) (*entity.QuestionAnswerSession, error)
}

type questionAnswerRepository struct{ db *gorm.DB }

func NewQuestionAnswerRepository(db *gorm.DB) IQuestionAnswerRepository {
	return &questionAnswerRepository{db: db}
}

func (r *questionAnswerRepository) CreateSession(ctx context.Context, tx ITransaction, session *entity.QuestionAnswerSession) error {
	db, ok := transactionDB(tx)
	if !ok || session == nil {
		return entity.ErrInvalidInput
	}
	if err := db.WithContext(ctx).Create(session).Error; err != nil {
		return entity.ErrAnswerSaveFailed
	}
	return nil
}

func (r *questionAnswerRepository) GetLatest(ctx context.Context, userID uuid.UUID) (*entity.QuestionAnswerSession, error) {
	var session entity.QuestionAnswerSession
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("answered_at DESC, created_at DESC").
		Preload("Answers.Choices").
		First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, entity.ErrInternal
	}
	return &session, nil
}
