package entity

import "github.com/google/uuid"

// RecommendationFeedbackSignal is a raw feedback aggregate returned by the
// repository. It intentionally contains no recommendation score or penalty.
type RecommendationFeedbackSignal struct {
	ArtistID uuid.UUID
	SongID   uuid.UUID
	Status   FeedbackStatus
	Count    int64
}
