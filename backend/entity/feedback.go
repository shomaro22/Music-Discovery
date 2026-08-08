package entity

import (
	"time"

	"github.com/google/uuid"
)

type FeedbackType string

const (
	FeedbackTypeLike  FeedbackType = "LIKE"
	FeedbackTypeMaybe FeedbackType = "MAYBE"
	FeedbackTypeSkip  FeedbackType = "SKIP"
)

func (value FeedbackType) IsValid() bool {
	return value == FeedbackTypeLike || value == FeedbackTypeMaybe || value == FeedbackTypeSkip
}

type Feedback struct {
	ID                     uuid.UUID            `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID                 uuid.UUID            `json:"user_id" gorm:"type:uuid;not null;index"`
	RecommendationDetailID uuid.UUID            `json:"recommendation_detail_id" gorm:"type:uuid;not null;uniqueIndex"`
	SongID                 uuid.UUID            `json:"song_id" gorm:"type:uuid;not null;index"`
	ArtistID               uuid.UUID            `json:"artist_id" gorm:"type:uuid;not null;index"`
	Type                   FeedbackType         `json:"feedback_type" gorm:"column:feedback_type;size:16;not null;index"`
	RecommendationDetail   RecommendationDetail `json:"-" gorm:"foreignKey:RecommendationDetailID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	CreatedAt              time.Time            `json:"created_at"`
	UpdatedAt              time.Time            `json:"updated_at"`
}

type UserArtistWeight struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null;uniqueIndex:uq_user_artist_weight"`
	ArtistID  uuid.UUID `json:"artist_id" gorm:"type:uuid;not null;uniqueIndex:uq_user_artist_weight"`
	Weight    float64   `json:"weight" gorm:"not null;default:0"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserSongWeight struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null;uniqueIndex:uq_user_song_weight"`
	SongID    uuid.UUID `json:"song_id" gorm:"type:uuid;not null;uniqueIndex:uq_user_song_weight"`
	Weight    float64   `json:"weight" gorm:"not null;default:0"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
