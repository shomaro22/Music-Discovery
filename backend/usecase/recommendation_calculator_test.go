package usecase

import (
	"github.com/google/uuid"
	"music-discovery/backend/entity"
	"testing"
)

func TestRecommendationCalculatorUsesMatchAndDiscoveryWeights(t *testing.T) {
	artistA, artistB := uuid.New(), uuid.New()
	songA, songB := uuid.New(), uuid.New()
	values := []entity.RecommendationSongCandidate{
		{Song: entity.Song{ID: songA, ArtistID: artistA}, Artist: entity.Artist{ID: artistA}, SongTags: []entity.Tag{{Slug: "happy", Category: "mood"}, {Slug: "rock", Category: "genre"}}},
		{Song: entity.Song{ID: songB, ArtistID: artistB}, Artist: entity.Artist{ID: artistB}, SongTags: []entity.Tag{{Slug: "calm", Category: "mood"}, {Slug: "jazz", Category: "genre"}}},
	}
	condition := RecommendationCondition{Mood: []string{"happy"}, Genres: []string{"rock"}, DiscoveryLevel: "high", GenreWeights: map[string]float64{}, ArtistWeights: map[uuid.UUID]float64{}, SongWeights: map[uuid.UUID]float64{}}
	calc := NewRecommendationCalculator(RecommendationScoreWeights{YourMatch: RecommendationTypeScoreWeights{Mood: 3, Element: 1.5, Atmosphere: 1.5, Genre: 3, Artist: 1, Song: 1, Discovery: .25}, NewFavorite: RecommendationTypeScoreWeights{Mood: 1.5, Element: 1, Atmosphere: 1, Genre: 2, Artist: 1, Song: 1, Discovery: 2}, TrySomethingNew: RecommendationTypeScoreWeights{Mood: 1, Element: 1, Atmosphere: 1, Genre: .5, Artist: .5, Song: .5, Discovery: 4}, ArtistRecency: .15, SongRecency: .3})
	match := calc.Calculate(condition, values, entity.RecommendationTypeYourMatch, map[uuid.UUID]int64{}, map[uuid.UUID]int64{})
	if match[0].Data.Song.ID != songA {
		t.Fatalf("your match should prefer matching song")
	}
	discovery := calc.Calculate(condition, values, entity.RecommendationTypeTrySomethingNew, map[uuid.UUID]int64{artistA: 2}, map[uuid.UUID]int64{songA: 2})
	if discovery[0].Data.Song.ID != songB {
		t.Fatalf("discovery should prefer unknown song")
	}
}

func TestRecommendationCalculatorAppliesFeedbackWeightsAndPenalty(t *testing.T) {
	artistLiked, artistSkipped := uuid.New(), uuid.New()
	songLiked, songSkipped := uuid.New(), uuid.New()
	values := []entity.RecommendationSongCandidate{
		{Song: entity.Song{ID: songLiked, ArtistID: artistLiked}, Artist: entity.Artist{ID: artistLiked}},
		{Song: entity.Song{ID: songSkipped, ArtistID: artistSkipped}, Artist: entity.Artist{ID: artistSkipped}},
	}
	weights := RecommendationScoreWeights{YourMatch: RecommendationTypeScoreWeights{Artist: 1, Song: 1}, ArtistRecency: .15, SongRecency: .3}
	condition := RecommendationCondition{
		GenreWeights: map[string]float64{}, ArtistWeights: map[uuid.UUID]float64{artistLiked: 2}, SongWeights: map[uuid.UUID]float64{songLiked: 1},
		ArtistFeedbackPenalties: map[uuid.UUID]float64{artistSkipped: 2}, SongFeedbackPenalties: map[uuid.UUID]float64{songSkipped: 2},
	}
	ranked := NewRecommendationCalculator(weights).Calculate(condition, values, entity.RecommendationTypeYourMatch, map[uuid.UUID]int64{}, map[uuid.UUID]int64{})
	if ranked[0].Data.Song.ID != songLiked {
		t.Fatalf("positive feedback weights should rank the liked song first")
	}
	if ranked[1].Score.FeedbackPenalty != 4 {
		t.Fatalf("feedback penalty=%v want=4", ranked[1].Score.FeedbackPenalty)
	}
}
