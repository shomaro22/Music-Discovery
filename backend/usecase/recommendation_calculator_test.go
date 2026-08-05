package usecase

import (
	"github.com/google/uuid"
	"math"
	"music-discovery/backend/entity"
	"testing"
)

func mustCalculate(t *testing.T, calculator IRecommendationCalculator, condition RecommendationCondition, candidates []entity.RecommendationSongCandidate, kind entity.RecommendationType, artistCounts map[uuid.UUID]int64, songCounts map[uuid.UUID]int64) []SongCandidate {
	t.Helper()
	result, err := calculator.Calculate(condition, candidates, kind, artistCounts, songCounts)
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}
	return result
}

func TestRecommendationCalculatorUsesMatchAndDiscoveryWeights(t *testing.T) {
	artistA, artistB := uuid.New(), uuid.New()
	songA, songB := uuid.New(), uuid.New()
	values := []entity.RecommendationSongCandidate{
		{Song: entity.Song{ID: songA, ArtistID: artistA}, Artist: entity.Artist{ID: artistA}, SongTags: []entity.Tag{{Slug: "happy", Category: "mood"}, {Slug: "rock", Category: "genre"}}},
		{Song: entity.Song{ID: songB, ArtistID: artistB}, Artist: entity.Artist{ID: artistB}, SongTags: []entity.Tag{{Slug: "calm", Category: "mood"}, {Slug: "jazz", Category: "genre"}}},
	}
	condition := RecommendationCondition{Mood: []string{"happy"}, Genres: []string{"rock"}, DiscoveryLevel: "high", GenreWeights: map[string]float64{}, ArtistWeights: map[uuid.UUID]float64{}, SongWeights: map[uuid.UUID]float64{}}
	calc := NewRecommendationCalculator(RecommendationScoreWeights{YourMatch: RecommendationTypeScoreWeights{Mood: 3, Element: 1.5, Atmosphere: 1.5, Genre: 3, Artist: 1, Song: 1, Discovery: .25}, NewFavorite: RecommendationTypeScoreWeights{Mood: 1.5, Element: 1, Atmosphere: 1, Genre: 2, Artist: 1, Song: 1, Discovery: 2}, TrySomethingNew: RecommendationTypeScoreWeights{Mood: 1, Element: 1, Atmosphere: 1, Genre: .5, Artist: .5, Song: .5, Discovery: 4}, ArtistRecency: .15, GenreRecency: .3})
	match := mustCalculate(t, calc, condition, values, entity.RecommendationTypeYourMatch, map[uuid.UUID]int64{}, map[uuid.UUID]int64{})
	if match[0].Data.Song.ID != songA {
		t.Fatalf("your match should prefer matching song")
	}
	discovery := mustCalculate(t, calc, condition, values, entity.RecommendationTypeTrySomethingNew, map[uuid.UUID]int64{artistA: 2}, map[uuid.UUID]int64{songA: 2})
	if discovery[0].Data.Song.ID != songB {
		t.Fatalf("discovery should prefer unknown song")
	}
}

func TestRecommendationCalculatorAppliesPersistedFeedbackWeights(t *testing.T) {
	artistLiked, artistSkipped := uuid.New(), uuid.New()
	songLiked, songSkipped := uuid.New(), uuid.New()
	values := []entity.RecommendationSongCandidate{
		{Song: entity.Song{ID: songLiked, ArtistID: artistLiked}, Artist: entity.Artist{ID: artistLiked}},
		{Song: entity.Song{ID: songSkipped, ArtistID: artistSkipped}, Artist: entity.Artist{ID: artistSkipped}},
	}
	weights := RecommendationScoreWeights{YourMatch: RecommendationTypeScoreWeights{Artist: 1, Song: 1}, ArtistRecency: .15, GenreRecency: .3}
	condition := RecommendationCondition{
		GenreWeights: map[string]float64{}, ArtistWeights: map[uuid.UUID]float64{artistLiked: 2}, SongWeights: map[uuid.UUID]float64{songLiked: 1},
	}
	ranked := mustCalculate(t, NewRecommendationCalculator(weights), condition, values, entity.RecommendationTypeYourMatch, map[uuid.UUID]int64{}, map[uuid.UUID]int64{})
	if ranked[0].Data.Song.ID != songLiked {
		t.Fatalf("positive feedback weights should rank the liked song first")
	}
}

func TestRecommendationCalculatorCountsSongGenreOnlyOnce(t *testing.T) {
	artistID := uuid.New()
	songID := uuid.New()
	candidate := entity.RecommendationSongCandidate{
		Song:   entity.Song{ID: songID, ArtistID: artistID},
		Artist: entity.Artist{ID: artistID},
		SongTags: []entity.Tag{
			{Slug: "rock", Name: "rock", Category: "genre"},
			{Slug: "rock", Name: "Rock", Category: "genre"},
		},
		ArtistTags: []entity.Tag{
			{Slug: "rock", Name: "Rock", Category: "genre"},
		},
	}
	condition := RecommendationCondition{
		Genres:             []string{"rock"},
		GenreWeights:       map[string]float64{"rock": 1},
		ArtistWeights:      map[uuid.UUID]float64{},
		SongWeights:        map[uuid.UUID]float64{},
		RecentArtistCounts: map[uuid.UUID]int64{},
		RecentGenreCounts:  map[string]int64{"rock": 2},
	}
	weights := RecommendationScoreWeights{
		YourMatch:    RecommendationTypeScoreWeights{Genre: 1},
		GenreRecency: .3,
	}

	ranked := mustCalculate(t, NewRecommendationCalculator(weights),
		condition,
		[]entity.RecommendationSongCandidate{candidate},
		entity.RecommendationTypeYourMatch,
		map[uuid.UUID]int64{},
		map[uuid.UUID]int64{},
	)

	if got := ranked[0].Score.Genre; got != 3 {
		t.Fatalf("genre score=%v want=3; duplicate song tags and artist tags must not be added", got)
	}
	if got := ranked[0].Score.RecencyPenalty; got != .6 {
		t.Fatalf("recency penalty=%v want=0.6; the same genre must be penalized once", got)
	}
}

func TestRecommendationCalculatorIgnoresArtistGenreTags(t *testing.T) {
	artistID := uuid.New()
	songID := uuid.New()
	candidate := entity.RecommendationSongCandidate{
		Song:       entity.Song{ID: songID, ArtistID: artistID},
		Artist:     entity.Artist{ID: artistID},
		ArtistTags: []entity.Tag{{Slug: "rock", Name: "Rock", Category: "genre"}},
	}
	condition := RecommendationCondition{
		Genres:             []string{"rock"},
		GenreWeights:       map[string]float64{"rock": 1},
		ArtistWeights:      map[uuid.UUID]float64{},
		SongWeights:        map[uuid.UUID]float64{},
		RecentArtistCounts: map[uuid.UUID]int64{},
		RecentGenreCounts:  map[string]int64{"rock": 2},
	}
	weights := RecommendationScoreWeights{
		YourMatch:    RecommendationTypeScoreWeights{Genre: 1},
		GenreRecency: .3,
	}

	ranked := mustCalculate(t, NewRecommendationCalculator(weights),
		condition,
		[]entity.RecommendationSongCandidate{candidate},
		entity.RecommendationTypeYourMatch,
		map[uuid.UUID]int64{},
		map[uuid.UUID]int64{},
	)

	if ranked[0].Score.Genre != 0 {
		t.Fatalf("artist genre tag must not contribute to genre score: %v", ranked[0].Score.Genre)
	}
	if ranked[0].Score.RecencyPenalty != 0 {
		t.Fatalf("artist genre tag must not contribute to genre recency: %v", ranked[0].Score.RecencyPenalty)
	}
}

func TestRecommendationCalculatorUsesPersistedElementAndAtmosphereWeights(t *testing.T) {
	artistA, artistB := uuid.New(), uuid.New()
	songA, songB := uuid.New(), uuid.New()
	values := []entity.RecommendationSongCandidate{
		{Song: entity.Song{ID: songA, ArtistID: artistA}, Artist: entity.Artist{ID: artistA}, SongTags: []entity.Tag{{Slug: "lyrics", Category: "element"}, {Slug: "night", Category: "atmosphere"}}},
		{Song: entity.Song{ID: songB, ArtistID: artistB}, Artist: entity.Artist{ID: artistB}},
	}
	condition := RecommendationCondition{
		GenreWeights: map[string]float64{}, ElementWeights: map[string]float64{"lyrics": 2}, AtmosphereWeights: map[string]float64{"night": 3},
		ArtistWeights: map[uuid.UUID]float64{}, SongWeights: map[uuid.UUID]float64{},
	}
	weights := RecommendationScoreWeights{YourMatch: RecommendationTypeScoreWeights{Element: 1, Atmosphere: 1}}
	ranked := mustCalculate(t, NewRecommendationCalculator(weights), condition, values, entity.RecommendationTypeYourMatch, map[uuid.UUID]int64{}, map[uuid.UUID]int64{})
	if ranked[0].Data.Song.ID != songA {
		t.Fatalf("persisted tag weights must affect ranking")
	}
	if ranked[0].Score.Element != 2 || ranked[0].Score.Atmosphere != 3 {
		t.Fatalf("unexpected persisted tag scores: %+v", ranked[0].Score)
	}
}

func TestRecommendationCalculatorUsesDeterministicTieBreakers(t *testing.T) {
	artistID := uuid.New()
	songSmall := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	songLarge := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	values := []entity.RecommendationSongCandidate{
		{Song: entity.Song{ID: songLarge, ArtistID: artistID}, Artist: entity.Artist{ID: artistID}},
		{Song: entity.Song{ID: songSmall, ArtistID: artistID}, Artist: entity.Artist{ID: artistID}},
	}
	condition := RecommendationCondition{GenreWeights: map[string]float64{}, ArtistWeights: map[uuid.UUID]float64{}, SongWeights: map[uuid.UUID]float64{}}
	ranked := mustCalculate(t, NewRecommendationCalculator(RecommendationScoreWeights{}), condition, values, entity.RecommendationTypeYourMatch, map[uuid.UUID]int64{}, map[uuid.UUID]int64{})
	if ranked[0].Data.Song.ID != songSmall {
		t.Fatalf("tie must be resolved by song id: got=%s", ranked[0].Data.Song.ID)
	}
}

func TestValidateRecommendationScoreWeightsRejectsNegativeConfig(t *testing.T) {
	weights := DefaultRecommendationScoreWeights()
	weights.YourMatch.Mood = -1
	if err := ValidateRecommendationScoreWeights(weights); err != entity.ErrInvalidWeightConfig {
		t.Fatalf("err=%v want=%v", err, entity.ErrInvalidWeightConfig)
	}
}

func TestRecommendationCalculatorAppliesNegativeGenreWeightToScoreAndPenalty(t *testing.T) {
	artistID, songID := uuid.New(), uuid.New()
	condition := RecommendationCondition{
		GenreWeights:  map[string]float64{"rock": -3},
		ArtistWeights: map[uuid.UUID]float64{},
		SongWeights:   map[uuid.UUID]float64{},
	}
	weights := RecommendationScoreWeights{YourMatch: RecommendationTypeScoreWeights{Genre: 1, FeedbackPenalty: 1}}
	ranked := mustCalculate(t, NewRecommendationCalculator(weights), condition, []entity.RecommendationSongCandidate{{
		Song: entity.Song{ID: songID, ArtistID: artistID}, Artist: entity.Artist{ID: artistID},
		SongTags: []entity.Tag{{Slug: "rock", Category: "genre"}},
	}}, entity.RecommendationTypeYourMatch, map[uuid.UUID]int64{}, map[uuid.UUID]int64{})
	if ranked[0].Score.Genre != -3 {
		t.Fatalf("genre score=%v want=-3", ranked[0].Score.Genre)
	}
	if ranked[0].Score.FeedbackPenalty <= 0 {
		t.Fatalf("negative genre weight must contribute to feedback penalty: %+v", ranked[0].Score)
	}
}

func TestRecommendationCalculatorRejectsInvalidRecommendationType(t *testing.T) {
	_, err := NewRecommendationCalculator(RecommendationScoreWeights{}).Calculate(RecommendationCondition{}, nil, entity.RecommendationType("UNKNOWN"), nil, nil)
	if err != entity.ErrInvalidRecommendationType {
		t.Fatalf("err=%v want=%v", err, entity.ErrInvalidRecommendationType)
	}
}

func TestValidateRecommendationScoreWeightsRejectsInvalidRecencyWeights(t *testing.T) {
	tests := []struct {
		name   string
		artist float64
		genre  float64
	}{
		{name: "artist NaN", artist: math.NaN()},
		{name: "artist positive infinity", artist: math.Inf(1)},
		{name: "genre NaN", genre: math.NaN()},
		{name: "genre negative infinity", genre: math.Inf(-1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			weights := DefaultRecommendationScoreWeights()
			weights.ArtistRecency = test.artist
			weights.GenreRecency = test.genre
			if err := ValidateRecommendationScoreWeights(weights); err != entity.ErrInvalidWeightConfig {
				t.Fatalf("err=%v want=%v", err, entity.ErrInvalidWeightConfig)
			}
		})
	}
}

func TestRecommendationCalculatorDoesNotAddExactAndRelatedGenreMatchesTogether(t *testing.T) {
	artistID, songID := uuid.New(), uuid.New()
	condition := RecommendationCondition{
		Genres:        []string{"rock", "alternative"},
		GenreWeights:  map[string]float64{"rock": 0},
		ArtistWeights: map[uuid.UUID]float64{},
		SongWeights:   map[uuid.UUID]float64{},
	}
	candidate := entity.RecommendationSongCandidate{
		Song:     entity.Song{ID: songID, ArtistID: artistID},
		Artist:   entity.Artist{ID: artistID},
		SongTags: []entity.Tag{{Slug: "rock", Category: "genre"}},
	}
	weights := RecommendationScoreWeights{YourMatch: RecommendationTypeScoreWeights{Genre: 1}}
	ranked := mustCalculate(t, NewRecommendationCalculator(weights), condition,
		[]entity.RecommendationSongCandidate{candidate},
		entity.RecommendationTypeYourMatch,
		map[uuid.UUID]int64{}, map[uuid.UUID]int64{},
	)
	if got := ranked[0].Score.Genre; got != 2 {
		t.Fatalf("genre score=%v want=2; exact and related matches must not be added together", got)
	}
}

func TestRecommendationCalculatorAddsMediumGenreContactOnlyOnce(t *testing.T) {
	artistID, songID := uuid.New(), uuid.New()
	condition := RecommendationCondition{
		Genres:            []string{"rock", "pop"},
		DiscoveryLevel:    "medium",
		GenreWeights:      map[string]float64{},
		ArtistWeights:     map[uuid.UUID]float64{},
		SongWeights:       map[uuid.UUID]float64{},
		RecommendedGenres: map[string]bool{},
	}
	candidate := entity.RecommendationSongCandidate{
		Song:   entity.Song{ID: songID, ArtistID: artistID},
		Artist: entity.Artist{ID: artistID},
		SongTags: []entity.Tag{
			{Slug: "alternative", Category: "genre"},
			{Slug: "electronic", Category: "genre"},
		},
	}
	weights := RecommendationScoreWeights{TrySomethingNew: RecommendationTypeScoreWeights{Discovery: 1}}
	ranked := mustCalculate(t, NewRecommendationCalculator(weights), condition,
		[]entity.RecommendationSongCandidate{candidate},
		entity.RecommendationTypeTrySomethingNew,
		map[uuid.UUID]int64{artistID: 1},
		map[uuid.UUID]int64{songID: 1},
	)
	if got := ranked[0].Score.Discovery; got != 4 {
		t.Fatalf("discovery score=%v want=4; medium genre contact must be added once", got)
	}
}
