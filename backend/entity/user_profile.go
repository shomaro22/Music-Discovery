package entity

import (
	"time"

	"github.com/google/uuid"
)

type UserProfile struct {
	ID             uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID         uuid.UUID `json:"user_id" gorm:"type:uuid;not null;uniqueIndex"`
	DiscoveryLevel string    `json:"discovery_level" gorm:"size:64;not null"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type UserGenreWeight struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProfileID uuid.UUID `json:"profile_id" gorm:"type:uuid;not null;index;uniqueIndex:uq_profile_genre"`
	ChoiceID  uuid.UUID `json:"choice_id" gorm:"type:uuid;not null;uniqueIndex:uq_profile_genre"`
	Weight    float64   `json:"weight" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserElementWeight struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProfileID uuid.UUID `json:"profile_id" gorm:"type:uuid;not null;index;uniqueIndex:uq_profile_element"`
	ChoiceID  uuid.UUID `json:"choice_id" gorm:"type:uuid;not null;uniqueIndex:uq_profile_element"`
	Weight    float64   `json:"weight" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserAtmosphereWeight struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProfileID uuid.UUID `json:"profile_id" gorm:"type:uuid;not null;index;uniqueIndex:uq_profile_atmosphere"`
	ChoiceID  uuid.UUID `json:"choice_id" gorm:"type:uuid;not null;uniqueIndex:uq_profile_atmosphere"`
	Weight    float64   `json:"weight" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
