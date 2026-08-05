package usecase

import (
	"testing"

	"github.com/google/uuid"
	"music-discovery/backend/entity"
)

func discoveryCandidate(artistID, songID uuid.UUID, genres ...string) SongCandidate {
	tags := make([]entity.Tag, 0, len(genres)+1)
	for _, genre := range genres {
		tags = append(tags, entity.Tag{Slug: genre, Category: "genre"})
	}
	tags = append(tags, entity.Tag{Slug: "calm", Category: "mood"})
	return SongCandidate{
		Data: entity.RecommendationSongCandidate{
			Song:     entity.Song{ID: songID, ArtistID: artistID},
			Artist:   entity.Artist{ID: artistID},
			SongTags: tags,
		},
		Score: RecommendationScore{Mood: 1, Total: 10},
	}
}

func selectionUsecase() *recommendationUsecase {
	return &recommendationUsecase{config: RecommendationConfig{MinimumMatchScore: 1}}
}

func TestTrySomethingNewLowPrefersUnknownArtistInSameGenre(t *testing.T) {
	knownArtist, unknownArtist := uuid.New(), uuid.New()
	condition := RecommendationCondition{
		Genres:            []string{"rock"},
		DiscoveryLevel:    "low",
		ArtistWeights:     map[uuid.UUID]float64{knownArtist: 1},
		RecommendedGenres: map[string]bool{"rock": true},
	}
	ranked := []SongCandidate{
		discoveryCandidate(knownArtist, uuid.New(), "jazz"),
		discoveryCandidate(unknownArtist, uuid.New(), "rock"),
	}
	got, _, ok := selectionUsecase().pickCandidate(ranked, entity.RecommendationTypeTrySomethingNew, condition, map[uuid.UUID]int64{}, map[uuid.UUID]int64{}, map[uuid.UUID]bool{}, map[uuid.UUID]bool{}, false)
	if !ok || got.Data.Artist.ID != unknownArtist {
		t.Fatalf("low discovery selected artist=%v, want unknown artist in same genre", got.Data.Artist.ID)
	}
}

func TestTrySomethingNewMediumPrefersNearbyGenre(t *testing.T) {
	nearArtist, unrelatedArtist := uuid.New(), uuid.New()
	condition := RecommendationCondition{
		Genres:            []string{"rock"},
		DiscoveryLevel:    "medium",
		ArtistWeights:     map[uuid.UUID]float64{},
		RecommendedGenres: map[string]bool{"rock": true},
	}
	ranked := []SongCandidate{
		discoveryCandidate(unrelatedArtist, uuid.New(), "jazz"),
		discoveryCandidate(nearArtist, uuid.New(), "alternative"),
	}
	got, _, ok := selectionUsecase().pickCandidate(ranked, entity.RecommendationTypeTrySomethingNew, condition, map[uuid.UUID]int64{}, map[uuid.UUID]int64{}, map[uuid.UUID]bool{}, map[uuid.UUID]bool{}, false)
	if !ok || got.Data.Artist.ID != nearArtist {
		t.Fatalf("medium discovery selected artist=%v, want nearby genre", got.Data.Artist.ID)
	}
}

func TestTrySomethingNewHighUsesRecommendationHistoryForUnknownGenre(t *testing.T) {
	seenGenreArtist, newGenreArtist := uuid.New(), uuid.New()
	condition := RecommendationCondition{
		Genres:            []string{"rock"},
		DiscoveryLevel:    "high",
		ArtistWeights:     map[uuid.UUID]float64{},
		RecommendedGenres: map[string]bool{"jazz": true, "rock": true},
	}
	ranked := []SongCandidate{
		discoveryCandidate(seenGenreArtist, uuid.New(), "jazz"),
		discoveryCandidate(newGenreArtist, uuid.New(), "classical"),
	}
	got, _, ok := selectionUsecase().pickCandidate(ranked, entity.RecommendationTypeTrySomethingNew, condition, map[uuid.UUID]int64{}, map[uuid.UUID]int64{}, map[uuid.UUID]bool{}, map[uuid.UUID]bool{}, false)
	if !ok || got.Data.Artist.ID != newGenreArtist {
		t.Fatalf("high discovery selected artist=%v, want artist with unseen genre", got.Data.Artist.ID)
	}
}

func TestTrySomethingNewRelaxesConditionsButKeepsContact(t *testing.T) {
	artist := uuid.New()
	condition := RecommendationCondition{
		Genres:            []string{"rock"},
		DiscoveryLevel:    "high",
		ArtistWeights:     map[uuid.UUID]float64{artist: 1},
		RecommendedGenres: map[string]bool{"rock": true},
	}
	candidate := discoveryCandidate(artist, uuid.New(), "jazz")
	artistCounts := map[uuid.UUID]int64{artist: 1}
	got, stage, ok := selectionUsecase().pickCandidate([]SongCandidate{candidate}, entity.RecommendationTypeTrySomethingNew, condition, artistCounts, map[uuid.UUID]int64{}, map[uuid.UUID]bool{}, map[uuid.UUID]bool{}, false)
	if !ok || got.Data.Song.ID != candidate.Data.Song.ID {
		t.Fatal("expected fallback to select the candidate")
	}
	if stage == 0 {
		t.Fatal("expected a fallback stage greater than zero")
	}

	candidate.Score.Mood = 0
	if _, _, ok := selectionUsecase().pickCandidate([]SongCandidate{candidate}, entity.RecommendationTypeTrySomethingNew, condition, artistCounts, map[uuid.UUID]int64{}, map[uuid.UUID]bool{}, map[uuid.UUID]bool{}, false); ok {
		t.Fatal("fallback must not remove the minimum contact requirement")
	}
}

func TestTrySomethingNewUsesRecommendationHistoryForUnknownArtist(t *testing.T) {
	seenArtist, unseenArtist := uuid.New(), uuid.New()
	condition := RecommendationCondition{
		Genres:            []string{"rock"},
		DiscoveryLevel:    "low",
		ArtistWeights:     map[uuid.UUID]float64{seenArtist: -0.5},
		RecommendedGenres: map[string]bool{"rock": true},
	}
	ranked := []SongCandidate{
		discoveryCandidate(seenArtist, uuid.New(), "rock"),
		discoveryCandidate(unseenArtist, uuid.New(), "rock"),
	}
	artistCounts := map[uuid.UUID]int64{seenArtist: 1}
	got, _, ok := selectionUsecase().pickCandidate(ranked, entity.RecommendationTypeTrySomethingNew, condition, artistCounts, map[uuid.UUID]int64{}, map[uuid.UUID]bool{}, map[uuid.UUID]bool{}, false)
	if !ok || got.Data.Artist.ID != unseenArtist {
		t.Fatalf("selected artist=%v, want artist absent from recommendation history", got.Data.Artist.ID)
	}
}

func TestNearbyGenresAreSymmetricAndCoverQuestionGenres(t *testing.T) {
	genres := []string{"pop", "rock", "alternative", "metal", "hip-hop", "jazz", "electronic", "classical", "anime", "k-pop"}
	for _, genre := range genres {
		neighbors, ok := nearbyGenres[genre]
		if !ok {
			t.Fatalf("nearby genre table does not contain %q", genre)
		}
		for neighbor := range neighbors {
			if !nearbyGenres[neighbor][genre] {
				t.Fatalf("genre relation must be symmetric: %s -> %s", genre, neighbor)
			}
		}
	}
}

func TestTrySomethingNewNeverSelectsCurrentRecommendationArtist(t *testing.T) {
	selectedArtist := uuid.New()
	otherArtist := uuid.New()
	condition := RecommendationCondition{
		Genres:            []string{"rock"},
		DiscoveryLevel:    "high",
		ArtistWeights:     map[uuid.UUID]float64{},
		RecommendedGenres: map[string]bool{"rock": true, "jazz": true},
	}
	ranked := []SongCandidate{
		discoveryCandidate(selectedArtist, uuid.New(), "jazz"),
		discoveryCandidate(otherArtist, uuid.New(), "classical"),
	}
	selectedArtists := map[uuid.UUID]bool{selectedArtist: true}

	got, _, ok := selectionUsecase().pickCandidate(
		ranked,
		entity.RecommendationTypeTrySomethingNew,
		condition,
		map[uuid.UUID]int64{},
		map[uuid.UUID]int64{},
		map[uuid.UUID]bool{},
		selectedArtists,
		false,
	)
	if !ok {
		t.Fatal("expected a candidate from another artist")
	}
	if got.Data.Artist.ID != otherArtist {
		t.Fatalf("selected artist=%v want=%v", got.Data.Artist.ID, otherArtist)
	}
}
