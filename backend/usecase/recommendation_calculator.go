package usecase

import (
	"math"
	"sort"
	"strings"

	"github.com/google/uuid"
	"music-discovery/backend/entity"
)

type RecommendationCondition struct {
	UserID              uuid.UUID
	Mood                []string
	Elements            []string
	Atmospheres         []string
	Genres              []string
	DiscoveryLevel      string
	DiscoveryAdjustment float64
	GenreWeights        map[string]float64
	ElementWeights      map[string]float64
	AtmosphereWeights   map[string]float64
	ArtistWeights       map[uuid.UUID]float64
	SongWeights         map[uuid.UUID]float64
	RecommendedGenres   map[string]bool
	RecentArtistCounts  map[uuid.UUID]int64
	RecentGenreCounts   map[string]int64
	ExcludedSongIDs     map[uuid.UUID]bool
}

type RecommendationScoreWeights struct {
	YourMatch       RecommendationTypeScoreWeights
	NewFavorite     RecommendationTypeScoreWeights
	TrySomethingNew RecommendationTypeScoreWeights
	DailyMatch      RecommendationTypeScoreWeights
	DailyChallenge  RecommendationTypeScoreWeights
	ArtistRecency   float64
	GenreRecency    float64
}

type RecommendationTypeScoreWeights struct {
	Mood            float64
	Element         float64
	Atmosphere      float64
	Genre           float64
	Artist          float64
	Song            float64
	Discovery       float64
	Novelty         float64
	FeedbackPenalty float64
	RecencyPenalty  float64
}

type RecommendationConfig struct {
	CandidateLimit      int
	DuplicateDays       int
	RecencyDays         int
	GenerationWindow    int
	LockTTLSeconds      int
	AlgorithmVersion    string
	WeightConfigVersion string
	MinimumMatchScore   float64
	ScoreWeights        RecommendationScoreWeights
}

type RecommendationScore struct {
	Mood            float64 `json:"mood_score"`
	Element         float64 `json:"element_score"`
	Atmosphere      float64 `json:"atmosphere_score"`
	Genre           float64 `json:"genre_score"`
	Artist          float64 `json:"artist_score"`
	Song            float64 `json:"song_score"`
	Discovery       float64 `json:"discovery_score"`
	Novelty         float64 `json:"novelty_score"`
	FeedbackPenalty float64 `json:"feedback_penalty"`
	RecencyPenalty  float64 `json:"recency_penalty"`
	Total           float64 `json:"total_score"`
}

type SongCandidate struct {
	Data  entity.RecommendationSongCandidate
	Score RecommendationScore
}

type IRecommendationCalculator interface {
	Calculate(RecommendationCondition, []entity.RecommendationSongCandidate, entity.RecommendationType, map[uuid.UUID]int64, map[uuid.UUID]int64) []SongCandidate
}

type recommendationCalculator struct{ weights RecommendationScoreWeights }

func NewRecommendationCalculator(weights RecommendationScoreWeights) IRecommendationCalculator {
	return &recommendationCalculator{weights: weights}
}

func ValidateRecommendationScoreWeights(weights RecommendationScoreWeights) error {
	values := []RecommendationTypeScoreWeights{weights.YourMatch, weights.NewFavorite, weights.TrySomethingNew, weights.DailyMatch, weights.DailyChallenge}
	for _, value := range values {
		for _, n := range []float64{value.Mood, value.Element, value.Atmosphere, value.Genre, value.Artist, value.Song, value.Discovery, value.Novelty, value.FeedbackPenalty, value.RecencyPenalty} {
			if math.IsNaN(n) || math.IsInf(n, 0) || n < 0 {
				return entity.ErrInvalidWeightConfig
			}
		}
	}
	if weights.ArtistRecency < 0 || weights.GenreRecency < 0 {
		return entity.ErrInvalidWeightConfig
	}
	return nil
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
	for _, tag := range append(append([]entity.Tag{}, candidate.SongTags...), candidate.ArtistTags...) {
		if v := normalized(tag.Slug); v != "" {
			result[v] = true
		}
		if v := normalized(tag.Name); v != "" {
			result[v] = true
		}
	}
	return result
}

func matches(values []string, tags map[string]bool) float64 {
	if len(values) == 0 {
		return 0
	}
	count := 0.0
	seen := map[string]bool{}
	for _, value := range values {
		key := normalized(value)
		if key != "" && !seen[key] {
			seen[key] = true
			if tags[key] {
				count++
			}
		}
	}
	return clamp(count/float64(len(seen))*10, 0, 10)
}

func songGenreKeys(candidate entity.RecommendationSongCandidate) []string {
	set := map[string]struct{}{}
	for _, tag := range candidate.SongTags {
		if normalized(tag.Category) != "genre" {
			continue
		}
		key := normalized(tag.Slug)
		if key == "" {
			key = normalized(tag.Name)
		}
		if key != "" {
			set[key] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for key := range set {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func scoreWeightsFor(weights RecommendationScoreWeights, kind entity.RecommendationType) RecommendationTypeScoreWeights {
	switch kind {
	case entity.RecommendationTypeYourMatch:
		return weights.YourMatch
	case entity.RecommendationTypeNewFavorite:
		return weights.NewFavorite
	case entity.RecommendationTypeTrySomethingNew:
		return weights.TrySomethingNew
	case entity.RecommendationTypeDailyMatchOne, entity.RecommendationTypeDailyMatchTwo:
		return weights.DailyMatch
	case entity.RecommendationTypeDailyChallenge:
		return weights.DailyChallenge
	default:
		return RecommendationTypeScoreWeights{}
	}
}

func scoreTagWeights(tags map[string]bool, values map[string]float64) float64 {
	total := 0.0
	count := 0.0
	for key, w := range values {
		if tags[normalized(key)] {
			total += w
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return clamp(total/count, -10, 10)
}

func (c *recommendationCalculator) Calculate(condition RecommendationCondition, values []entity.RecommendationSongCandidate, kind entity.RecommendationType, artistCounts map[uuid.UUID]int64, songCounts map[uuid.UUID]int64) []SongCandidate {
	return calculateWithWeights(condition, values, scoreWeightsFor(c.weights, kind), c.weights.ArtistRecency, c.weights.GenreRecency, artistCounts, songCounts)
}

func calculateWithWeights(condition RecommendationCondition, values []entity.RecommendationSongCandidate, weights RecommendationTypeScoreWeights, artistRecency, genreRecency float64, artistCounts map[uuid.UUID]int64, songCounts map[uuid.UUID]int64) []SongCandidate {
	result := make([]SongCandidate, 0, len(values))
	for _, value := range values {
		if condition.ExcludedSongIDs[value.Song.ID] {
			continue
		}
		tags := tagSet(value)
		genres := songGenreKeys(value)
		score := RecommendationScore{Mood: matches(condition.Mood, tags), Element: clamp(matches(condition.Elements, tags)+scoreTagWeights(tags, condition.ElementWeights), -10, 10), Atmosphere: clamp(matches(condition.Atmospheres, tags)+scoreTagWeights(tags, condition.AtmosphereWeights), -10, 10)}
		bestGenre := 0.0
		for _, genre := range genres {
			base := condition.GenreWeights[genre]
			for _, wanted := range condition.Genres {
				w := normalized(wanted)
				if genre == w {
					base += 2
				} else if genreRelated(w, genre) {
					base += 1
				}
			}
			if base > bestGenre {
				bestGenre = base
			}
		}
		score.Genre = clamp(bestGenre, -10, 10)
		score.Artist = clamp(condition.ArtistWeights[value.Artist.ID], -10, 10)
		score.Song = clamp(condition.SongWeights[value.Song.ID], -10, 10)
		score.FeedbackPenalty = clamp(math.Max(0, -score.Song)+math.Max(0, -score.Artist)*0.6+math.Max(0, -score.Genre)*0.25, 0, 10)
		knownArtist := artistCounts[value.Artist.ID] > 0
		knownSong := songCounts[value.Song.ID] > 0
		unknownGenre := true
		for _, g := range genres {
			if condition.RecommendedGenres[g] {
				unknownGenre = false
				break
			}
		}
		level := effectiveDiscoveryLevel(condition.DiscoveryLevel, condition.DiscoveryAdjustment)
		switch level {
		case "high":
			if unknownGenre {
				score.Discovery += 5
			}
			if !knownArtist {
				score.Discovery += 3
			}
			if !knownSong {
				score.Discovery += 2
			}
		case "medium":
			for _, g := range genres {
				for _, wanted := range condition.Genres {
					if genreRelated(normalized(wanted), g) {
						score.Discovery += 4
					}
				}
			}
			if !knownArtist {
				score.Discovery += 4
			}
			if !knownSong {
				score.Discovery += 2
			}
		default:
			if candidateMatchesGenres(value, condition.Genres) && !knownArtist {
				score.Discovery += 7
			}
			if !knownSong {
				score.Discovery += 2
			}
		}
		score.Discovery = clamp(score.Discovery, 0, 10)
		score.Novelty = clamp(novelty(songCounts[value.Song.ID], 5)+novelty(artistCounts[value.Artist.ID], 3), 0, 10)
		score.RecencyPenalty = float64(condition.RecentArtistCounts[value.Artist.ID]) * artistRecency
		for _, g := range genres {
			score.RecencyPenalty += float64(condition.RecentGenreCounts[g]) * genreRecency
		}
		score.RecencyPenalty = clamp(score.RecencyPenalty, 0, 10)
		score.Total = round2(score.Mood*weights.Mood + score.Element*weights.Element + score.Atmosphere*weights.Atmosphere + score.Genre*weights.Genre + score.Artist*weights.Artist + score.Song*weights.Song + score.Discovery*weights.Discovery + score.Novelty*weights.Novelty - score.FeedbackPenalty*weights.FeedbackPenalty - score.RecencyPenalty*maxOrOne(weights.RecencyPenalty))
		score.Total = clamp(score.Total, -100, 100)
		result = append(result, SongCandidate{Data: value, Score: score})
	}
	sort.SliceStable(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if a.Score.Total != b.Score.Total {
			return a.Score.Total > b.Score.Total
		}
		if a.Score.Novelty != b.Score.Novelty {
			return a.Score.Novelty > b.Score.Novelty
		}
		if a.Score.FeedbackPenalty != b.Score.FeedbackPenalty {
			return a.Score.FeedbackPenalty < b.Score.FeedbackPenalty
		}
		if artistCounts[a.Data.Artist.ID] != artistCounts[b.Data.Artist.ID] {
			return artistCounts[a.Data.Artist.ID] < artistCounts[b.Data.Artist.ID]
		}
		if songCounts[a.Data.Song.ID] != songCounts[b.Data.Song.ID] {
			return songCounts[a.Data.Song.ID] < songCounts[b.Data.Song.ID]
		}
		return a.Data.Song.ID.String() < b.Data.Song.ID.String()
	})
	return result
}

func novelty(count int64, max float64) float64 {
	if count <= 0 {
		return max
	}
	return max / float64(count+1)
}
func maxOrOne(v float64) float64 {
	if v == 0 {
		return 1
	}
	return v
}
func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
func round2(v float64) float64 { return math.Round(v*100) / 100 }

func effectiveDiscoveryLevel(level string, adjustment float64) string {
	current := normalized(level)
	index := 1
	if current == "low" {
		index = 0
	}
	if current == "high" {
		index = 2
	}
	if adjustment >= .75 {
		index++
	}
	if adjustment <= -.75 {
		index--
	}
	if index < 0 {
		index = 0
	}
	if index > 2 {
		index = 2
	}
	return []string{"low", "medium", "high"}[index]
}

func genreRelated(a, b string) bool {
	if a == b {
		return true
	}
	return nearbyGenres[a][b]
}
