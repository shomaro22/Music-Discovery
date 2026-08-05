package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type RecommendationType string

const (
	RecommendationTypeYourMatch       RecommendationType = "YOUR_MATCH"
	RecommendationTypeNewFavorite     RecommendationType = "NEW_FAVORITE"
	RecommendationTypeTrySomethingNew RecommendationType = "TRY_SOMETHING_NEW"
	RecommendationTypeDailyMatchOne   RecommendationType = "DAILY_MATCH_ONE"
	RecommendationTypeDailyMatchTwo   RecommendationType = "DAILY_MATCH_TWO"
	RecommendationTypeDailyChallenge  RecommendationType = "DAILY_CHALLENGE"
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
	ID                  uuid.UUID              `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID              uuid.UUID              `json:"user_id" gorm:"type:uuid;not null;index;uniqueIndex:uq_daily_discovery_day"`
	Source              RecommendationSource   `json:"source" gorm:"size:32;not null;index;uniqueIndex:uq_daily_discovery_day"`
	TargetDate          *time.Time             `json:"target_date,omitempty" gorm:"type:date;index;uniqueIndex:uq_daily_discovery_day"`
	RecommendedAt       time.Time              `json:"recommended_at" gorm:"not null;index"`
	AlgorithmVersion    string                 `json:"algorithm_version" gorm:"size:64;not null"`
	WeightConfigVersion string                 `json:"-" gorm:"size:64;not null;default:'weights-v1'"`
	Details             []RecommendationDetail `json:"details,omitempty" gorm:"foreignKey:RecommendationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	CreatedAt           time.Time              `json:"created_at"`
}

type RecommendationDetail struct {
	ID               uuid.UUID          `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RecommendationID uuid.UUID          `json:"recommendation_id" gorm:"type:uuid;not null;index;uniqueIndex:uq_recommendation_type;uniqueIndex:uq_recommendation_position;uniqueIndex:uq_recommendation_song"`
	SongID           uuid.UUID          `json:"song_id" gorm:"type:uuid;not null;index;uniqueIndex:uq_recommendation_song"`
	Type             RecommendationType `json:"recommendation_type" gorm:"column:recommendation_type;size:32;not null;uniqueIndex:uq_recommendation_type"`
	Score            float64            `json:"score" gorm:"not null"`
	ScoreBreakdown   json.RawMessage    `json:"-" gorm:"type:jsonb;not null;default:'{}'"`
	FallbackUsed     bool               `json:"-" gorm:"not null;default:false"`
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
