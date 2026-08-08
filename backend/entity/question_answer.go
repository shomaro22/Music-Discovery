package entity

import (
	"time"

	"github.com/google/uuid"
)

type QuestionAnswerSession struct {
	ID         uuid.UUID    `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID     uuid.UUID    `json:"user_id" gorm:"type:uuid;not null;index"`
	User       User         `json:"-" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;-:migration"`
	AnsweredAt time.Time    `json:"answered_at" gorm:"not null;index"`
	Answers    []UserAnswer `json:"answers,omitempty" gorm:"foreignKey:AnswerSessionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	CreatedAt  time.Time    `json:"created_at"`
}

type UserAnswer struct {
	ID              uuid.UUID             `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AnswerSessionID uuid.UUID             `json:"answer_session_id" gorm:"type:uuid;not null;index;uniqueIndex:uq_session_question"`
	AnswerSession   QuestionAnswerSession `json:"-" gorm:"foreignKey:AnswerSessionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	QuestionID      uuid.UUID             `json:"question_id" gorm:"type:uuid;not null;index;uniqueIndex:uq_session_question"`
	Question        Question              `json:"-" gorm:"foreignKey:QuestionID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Choices         []UserAnswerChoice    `json:"choices,omitempty" gorm:"foreignKey:UserAnswerID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	CreatedAt       time.Time             `json:"created_at"`
}

type UserAnswerChoice struct {
	ID               uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserAnswerID     uuid.UUID      `json:"user_answer_id" gorm:"type:uuid;not null;index;uniqueIndex:uq_user_answer_choice"`
	UserAnswer       UserAnswer     `json:"-" gorm:"foreignKey:UserAnswerID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	QuestionChoiceID uuid.UUID      `json:"question_choice_id" gorm:"type:uuid;not null;index;uniqueIndex:uq_user_answer_choice"`
	QuestionChoice   QuestionChoice `json:"-" gorm:"foreignKey:QuestionChoiceID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	CreatedAt        time.Time      `json:"created_at"`
}
