package entity

import (
	"time"

	"github.com/google/uuid"
)

type UserProfile struct {
	ID                uuid.UUID              `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID            uuid.UUID              `json:"user_id" gorm:"type:uuid;not null;uniqueIndex"`
	User              User                   `json:"-" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	DiscoveryLevel    string                 `json:"discovery_level" gorm:"size:64;not null"`
	GenreWeights      []UserGenreWeight      `json:"genre_weights,omitempty" gorm:"foreignKey:ProfileID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	ElementWeights    []UserElementWeight    `json:"element_weights,omitempty" gorm:"foreignKey:ProfileID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	AtmosphereWeights []UserAtmosphereWeight `json:"atmosphere_weights,omitempty" gorm:"foreignKey:ProfileID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

type UserGenreWeight struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProfileID      uuid.UUID      `json:"profile_id" gorm:"type:uuid;not null;index;uniqueIndex:uq_profile_genre"`
	Profile        UserProfile    `json:"-" gorm:"foreignKey:ProfileID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	ChoiceID       uuid.UUID      `json:"choice_id" gorm:"type:uuid;not null;uniqueIndex:uq_profile_genre"`
	Choice         QuestionChoice `json:"-" gorm:"foreignKey:ChoiceID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	QuestionWeight float64        `json:"question_weight" gorm:"not null;default:0"`
	FeedbackWeight float64        `json:"feedback_weight" gorm:"not null;default:0"`
	Weight         float64        `json:"weight" gorm:"not null;default:0"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type UserElementWeight struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProfileID      uuid.UUID      `json:"profile_id" gorm:"type:uuid;not null;index;uniqueIndex:uq_profile_element"`
	Profile        UserProfile    `json:"-" gorm:"foreignKey:ProfileID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	ChoiceID       uuid.UUID      `json:"choice_id" gorm:"type:uuid;not null;uniqueIndex:uq_profile_element"`
	Choice         QuestionChoice `json:"-" gorm:"foreignKey:ChoiceID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	QuestionWeight float64        `json:"question_weight" gorm:"not null;default:0"`
	FeedbackWeight float64        `json:"feedback_weight" gorm:"not null;default:0"`
	Weight         float64        `json:"weight" gorm:"not null;default:0"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type UserAtmosphereWeight struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProfileID      uuid.UUID      `json:"profile_id" gorm:"type:uuid;not null;index;uniqueIndex:uq_profile_atmosphere"`
	Profile        UserProfile    `json:"-" gorm:"foreignKey:ProfileID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	ChoiceID       uuid.UUID      `json:"choice_id" gorm:"type:uuid;not null;uniqueIndex:uq_profile_atmosphere"`
	Choice         QuestionChoice `json:"-" gorm:"foreignKey:ChoiceID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	QuestionWeight float64        `json:"question_weight" gorm:"not null;default:0"`
	FeedbackWeight float64        `json:"feedback_weight" gorm:"not null;default:0"`
	Weight         float64        `json:"weight" gorm:"not null;default:0"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}
