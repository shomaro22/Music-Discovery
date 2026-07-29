package usecase

import (
	"sort"
	"strings"

	"github.com/google/uuid"
	"music-discovery/backend/entity"
)

type RecommendationCondition struct {
	Mood                    []string
	Elements                []string
	Atmospheres             []string
	Genres                  []string
	DiscoveryLevel          string
	GenreWeights            map[string]float64
	ArtistWeights           map[uuid.UUID]float64
	SongWeights             map[uuid.UUID]float64
	ArtistFeedbackPenalties map[uuid.UUID]float64
	SongFeedbackPenalties   map[uuid.UUID]float64
	RecommendedGenres       map[string]bool
}

type RecommendationScoreWeights struct {
	YourMatch       RecommendationTypeScoreWeights
	NewFavorite     RecommendationTypeScoreWeights
	TrySomethingNew RecommendationTypeScoreWeights
	ArtistRecency   float64
	SongRecency     float64
}

type RecommendationTypeScoreWeights struct {
	Mood, Element, Atmosphere, Genre, Artist, Song, Discovery float64
}

type RecommendationConfig struct {
	CandidateLimit    int
	DuplicateDays     int
	GenerationWindow  int
	LockTTLSeconds    int
	AlgorithmVersion  string
	MinimumMatchScore float64
	ScoreWeights      RecommendationScoreWeights
}

type RecommendationScore struct{ Mood, Element, Atmosphere, Genre, Artist, Song, Discovery, FeedbackPenalty, RecencyPenalty, Total float64 }
type SongCandidate struct {
	Data  entity.RecommendationSongCandidate
	Score RecommendationScore
}
type IRecommendationCalculator interface {
	Calculate(condition RecommendationCondition, values []entity.RecommendationSongCandidate, kind entity.RecommendationType, artistCounts map[uuid.UUID]int64, songCounts map[uuid.UUID]int64) []SongCandidate
}
type recommendationCalculator struct{ weights RecommendationScoreWeights }

func NewRecommendationCalculator(weights RecommendationScoreWeights) IRecommendationCalculator {
	return &recommendationCalculator{weights: weights}
}

func normalized(value string) string {
	result := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "_", "-"))
	for _, prefix := range []string{"mood-", "element-", "atmosphere-", "genre-", "discovery-"} {
		result = strings.TrimPrefix(result, prefix)
	}
	return result
}
func tagSet(candidate entity.RecommendationSongCandidate) map[string]bool {
	result := map[string]bool{}
	for _, tag := range append(candidate.SongTags, candidate.ArtistTags...) {
		result[normalized(tag.Slug)] = true
		result[normalized(tag.Name)] = true
	}
	return result
}
func matches(values []string, tags map[string]bool) float64 {
	score := 0.0
	for _, value := range values {
		if tags[normalized(value)] {
			score++
		}
	}
	return score
}
func genreValues(candidate entity.RecommendationSongCandidate) []string {
	result := []string{}
	for _, tag := range append(candidate.SongTags, candidate.ArtistTags...) {
		if normalized(tag.Category) == "genre" {
			result = append(result, tag.Slug, tag.Name)
		}
	}
	return result
}
func scoreWeightsFor(weights RecommendationScoreWeights, kind entity.RecommendationType) RecommendationTypeScoreWeights {
	switch kind {
	case entity.RecommendationTypeYourMatch:
		return weights.YourMatch
	case entity.RecommendationTypeNewFavorite:
		return weights.NewFavorite
	default:
		return weights.TrySomethingNew
	}
}
func (c *recommendationCalculator) Calculate(condition RecommendationCondition, values []entity.RecommendationSongCandidate, kind entity.RecommendationType, artistCounts map[uuid.UUID]int64, songCounts map[uuid.UUID]int64) []SongCandidate {
	result := make([]SongCandidate, 0, len(values))
	weights := scoreWeightsFor(c.weights, kind)
	for _, value := range values {
		tags := tagSet(value)
		genres := genreValues(value)
		score := RecommendationScore{Mood: matches(condition.Mood, tags), Element: matches(condition.Elements, tags), Atmosphere: matches(condition.Atmospheres, tags)}
		for _, genre := range genres {
			key := normalized(genre)
			for _, wanted := range condition.Genres {
				if key == normalized(wanted) {
					score.Genre += 2
				}
			}
			score.Genre += condition.GenreWeights[key]
		}
		score.Artist = condition.ArtistWeights[value.Artist.ID]
		score.Song = condition.SongWeights[value.Song.ID]
		knownArtist := artistCounts[value.Artist.ID] > 0
		knownSong := songCounts[value.Song.ID] > 0
		switch normalized(condition.DiscoveryLevel) {
		case "high":
			if !knownArtist {
				score.Discovery += 2
			}
			if !knownSong {
				score.Discovery++
			}
		case "medium":
			if !knownArtist {
				score.Discovery += 1.5
			}
			if !knownSong {
				score.Discovery += .5
			}
		default:
			if !knownArtist {
				score.Discovery++
			}
		}
		score.FeedbackPenalty = condition.ArtistFeedbackPenalties[value.Artist.ID] + condition.SongFeedbackPenalties[value.Song.ID]
		score.RecencyPenalty = float64(artistCounts[value.Artist.ID])*c.weights.ArtistRecency + float64(songCounts[value.Song.ID])*c.weights.SongRecency
		score.Total = score.Mood*weights.Mood + score.Element*weights.Element + score.Atmosphere*weights.Atmosphere + score.Genre*weights.Genre + score.Artist*weights.Artist + score.Song*weights.Song + score.Discovery*weights.Discovery - score.FeedbackPenalty - score.RecencyPenalty
		result = append(result, SongCandidate{Data: value, Score: score})
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Score.Total == result[j].Score.Total {
			return result[i].Data.Song.ID.String() < result[j].Data.Song.ID.String()
		}
		return result[i].Score.Total > result[j].Score.Total
	})
	return result
}
