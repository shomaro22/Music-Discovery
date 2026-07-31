package entity

import (
	"time"

	"github.com/google/uuid"
)

type RecommendationType string

const (
	RecommendationTypeYourMatch       RecommendationType = "YOUR_MATCH"
	RecommendationTypeNewFavorite     RecommendationType = "NEW_FAVORITE"
	RecommendationTypeTrySomethingNew RecommendationType = "TRY_SOMETHING_NEW"
)

type RecommendationSource string

const (
	RecommendationSourceNormal         RecommendationSource = "NORMAL"
	RecommendationSourceDailyDiscovery RecommendationSource = "DAILY_DISCOVERY"
)

type FeedbackStatus string

const (
	FeedbackStatusLike  FeedbackStatus = "LIKE"
	FeedbackStatusMaybe FeedbackStatus = "MAYBE"
	FeedbackStatusSkip  FeedbackStatus = "SKIP"
)

type Recommendation struct {
	ID               uuid.UUID              `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID           uuid.UUID              `json:"user_id" gorm:"type:uuid;not null;index"`
	Source           RecommendationSource   `json:"source" gorm:"size:32;not null;index"`
	RecommendedAt    time.Time              `json:"recommended_at" gorm:"not null;index"`
	AlgorithmVersion string                 `json:"algorithm_version" gorm:"size:64;not null"`
	Details          []RecommendationDetail `json:"details,omitempty" gorm:"foreignKey:RecommendationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	CreatedAt        time.Time              `json:"created_at"`
}

type RecommendationDetail struct {
	ID               uuid.UUID          `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RecommendationID uuid.UUID          `json:"recommendation_id" gorm:"type:uuid;not null;index;uniqueIndex:uq_recommendation_type;uniqueIndex:uq_recommendation_position;uniqueIndex:uq_recommendation_song"`
	SongID           uuid.UUID          `json:"song_id" gorm:"type:uuid;not null;index;uniqueIndex:uq_recommendation_song"`
	Type             RecommendationType `json:"recommendation_type" gorm:"column:recommendation_type;size:32;not null;uniqueIndex:uq_recommendation_type"`
	Score            float64            `json:"score" gorm:"not null"`
	Position         int                `json:"position" gorm:"not null;uniqueIndex:uq_recommendation_position"`
	Reason           string             `json:"reason" gorm:"size:500;not null"`
	FeedbackStatus   *FeedbackStatus    `json:"feedback_status,omitempty" gorm:"-"`
	Song             Song               `json:"song" gorm:"foreignKey:SongID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

type RecommendationSongCandidate struct {
	Song       Song
	Artist     Artist
	SongTags   []Tag
	ArtistTags []Tag
}
