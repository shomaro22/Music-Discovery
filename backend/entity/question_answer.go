package entity

import (
	"time"

	"github.com/google/uuid"
)

type QuestionAnswerSession struct {
	ID         uuid.UUID    `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID     uuid.UUID    `json:"user_id" gorm:"type:uuid;not null;index"`
	AnsweredAt time.Time    `json:"answered_at" gorm:"not null;index"`
	Answers    []UserAnswer `json:"answers,omitempty" gorm:"foreignKey:AnswerSessionID"`
	CreatedAt  time.Time    `json:"created_at"`
}

type UserAnswer struct {
	ID              uuid.UUID          `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AnswerSessionID uuid.UUID          `json:"answer_session_id" gorm:"type:uuid;not null;index;uniqueIndex:uq_session_question"`
	QuestionID      uuid.UUID          `json:"question_id" gorm:"type:uuid;not null;index;uniqueIndex:uq_session_question"`
	Choices         []UserAnswerChoice `json:"choices,omitempty" gorm:"foreignKey:UserAnswerID"`
	CreatedAt       time.Time          `json:"created_at"`
}

type UserAnswerChoice struct {
	ID               uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserAnswerID     uuid.UUID `json:"user_answer_id" gorm:"type:uuid;not null;index;uniqueIndex:uq_user_answer_choice"`
	QuestionChoiceID uuid.UUID `json:"question_choice_id" gorm:"type:uuid;not null;index;uniqueIndex:uq_user_answer_choice"`
	CreatedAt        time.Time `json:"created_at"`
}
