package entity

import (
	"time"

	"github.com/google/uuid"
)

// Admin read models aggregate persisted entities for application queries.
// They intentionally carry no HTTP/JSON concerns.
type ArtistListItem struct {
	Artist Artist
	Tags   []Tag
}

type TagSummary struct {
	Tag             Tag
	ArtistCount     int64
	SongCount       int64
	UserWeightCount int64
}

type SongMetrics struct {
	RecommendationCount int64
	LikeCount           int64
	MaybeCount          int64
	SkipCount           int64
}

type UserMetrics struct {
	ProfileStatus       string
	RecommendationCount int64
	FeedbackCount       int64
	LastLoginAt         *time.Time
}

type ArtistDetail struct {
	Artist    Artist
	Tags      []Tag
	SongCount int64
}

type SongTagDetail struct {
	Tag    Tag
	Weight float64
}

type SongDetail struct {
	Song    Song
	Artist  Artist
	Tags    []SongTagDetail
	Metrics SongMetrics
}

type FeedbackTarget struct {
	Detail         RecommendationDetail
	Recommendation Recommendation
	Song           Song
	Artist         Artist
	Tags           []Tag
}

type SpotifyCandidateSaveResult struct {
	Created      int
	AutoApproved int
	Pending      int
	Skipped      int
}

type RateLimitResult struct {
	Allowed    bool
	Remaining  float64
	RetryAfter time.Duration
}

type RecommendationRecencyCounts struct {
	ArtistCounts map[uuid.UUID]int64
	GenreCounts  map[string]int64
}
