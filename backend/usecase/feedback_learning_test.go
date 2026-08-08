package usecase

import (
	"testing"
	"time"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
)

func TestCalculateLearningSnapshotUsesConfiguredDeltasAndClampsTotalWeight(t *testing.T) {
	profileID := uuid.New()
	choiceID := uuid.New()
	artistID := uuid.New()
	songID := uuid.New()
	now := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	u := &feedbackUsecase{config: FeedbackConfig{
		Like:                       FeedbackDelta{Genre: 3, Artist: 2, Song: 3, Profile: 2, Discovery: 1},
		MinimumWeight:              -10,
		MaximumWeight:              10,
		MinimumDiscoveryAdjustment: -2,
		MaximumDiscoveryAdjustment: 2,
	}}
	profile := &entity.UserProfile{
		ID: profileID,
		GenreWeights: []entity.UserGenreWeight{{
			ID: uuid.New(), ProfileID: profileID, ChoiceID: choiceID,
			QuestionWeight: 9, FeedbackWeight: 0, Weight: 9, CreatedAt: now,
		}},
	}
	sources := []entity.FeedbackLearningSource{
		{FeedbackID: uuid.New(), FeedbackType: entity.FeedbackTypeLike, ArtistID: artistID, SongID: songID, Exploration: true, Tags: []entity.FeedbackLearningTag{{ChoiceID: choiceID, Category: "genre"}}},
	}

	snapshot := u.calculateLearningSnapshot(uuid.New(), profile, sources, now)

	if got := snapshot.GenreWeights[0].FeedbackWeight; got != 3 {
		t.Fatalf("feedback_weight=%v want=3", got)
	}
	if got := snapshot.GenreWeights[0].Weight; got != 10 {
		t.Fatalf("combined weight=%v want=10", got)
	}
	if got := snapshot.ArtistWeights[0].Weight; got != 2 {
		t.Fatalf("artist weight=%v want=2", got)
	}
	if got := snapshot.SongWeights[0].Weight; got != 3 {
		t.Fatalf("song weight=%v want=3", got)
	}
	if got := snapshot.DiscoveryAdjustment; got != 1 {
		t.Fatalf("discovery adjustment=%v want=1", got)
	}
}

func TestMergeExistingFeedbackWeightsPreservesFeedbackAndClampsCombinedWeight(t *testing.T) {
	profileID := uuid.New()
	choiceID := uuid.New()
	profile := &entity.UserProfile{
		ID: profileID,
		GenreWeights: []entity.UserGenreWeight{{
			ProfileID: profileID, ChoiceID: choiceID,
			QuestionWeight: 0, FeedbackWeight: 10, Weight: 10,
		}},
	}
	values := []entity.UserGenreWeight{{
		ProfileID: profileID, ChoiceID: choiceID,
		QuestionWeight: 1, Weight: 1,
	}}

	mergeExistingFeedbackWeights(profile, values, nil, nil, -10, 10)

	if got := values[0].FeedbackWeight; got != 10 {
		t.Fatalf("feedback_weight=%v want=10", got)
	}
	if got := values[0].Weight; got != 10 {
		t.Fatalf("combined weight=%v want=10", got)
	}
}
