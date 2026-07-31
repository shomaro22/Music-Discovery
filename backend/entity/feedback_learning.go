package entity

import "github.com/google/uuid"

type FeedbackLearningTag struct {
	ChoiceID uuid.UUID
	Category string
}

type FeedbackLearningSource struct {
	FeedbackID   uuid.UUID
	FeedbackType FeedbackType
	ArtistID     uuid.UUID
	SongID       uuid.UUID
	Exploration  bool
	Tags         []FeedbackLearningTag
}

type FeedbackLearningSnapshot struct {
	ArtistWeights       []UserArtistWeight
	SongWeights         []UserSongWeight
	GenreWeights        []UserGenreWeight
	ElementWeights      []UserElementWeight
	AtmosphereWeights   []UserAtmosphereWeight
	DiscoveryAdjustment float64
}
