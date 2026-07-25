package entity

import (
	"time"

	"github.com/google/uuid"
)

type QuestionAnswerType string

const (
	QuestionAnswerTypeSingle   QuestionAnswerType = "single"
	QuestionAnswerTypeMultiple QuestionAnswerType = "multiple"
)

const (
	QuestionCodeMood           = "MOOD"
	QuestionCodeElement        = "ELEMENT"
	QuestionCodeAtmosphere     = "ATMOSPHERE"
	QuestionCodeGenre          = "GENRE"
	QuestionCodeDiscoveryLevel = "DISCOVERY_LEVEL"
)

type Question struct {
	ID            uuid.UUID          `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Code          string             `json:"code" gorm:"size:64;not null;uniqueIndex"`
	Title         string             `json:"title" gorm:"size:200;not null"`
	AnswerType    QuestionAnswerType `json:"answer_type" gorm:"size:16;not null"`
	MinSelections int                `json:"min_selections" gorm:"not null"`
	MaxSelections *int               `json:"max_selections,omitempty"`
	SortOrder     int                `json:"sort_order" gorm:"not null;uniqueIndex"`
	IsActive      bool               `json:"is_active" gorm:"not null;default:true;index"`
	Choices       []QuestionChoice   `json:"choices,omitempty"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

type QuestionChoice struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	QuestionID uuid.UUID `json:"question_id" gorm:"type:uuid;not null;index;uniqueIndex:uq_question_choice_code"`
	Code       string    `json:"code" gorm:"size:96;not null;uniqueIndex;uniqueIndex:uq_question_choice_code"`
	Label      string    `json:"label" gorm:"size:100;not null"`
	SortOrder  int       `json:"sort_order" gorm:"not null"`
	IsActive   bool      `json:"is_active" gorm:"not null;default:true;index"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
