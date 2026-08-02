package usecase

const (
	DailyDiscoveryMinimumMatchScore   = 2.0
	DailyDiscoveryMinimumContactScore = 1.0
)

type DailyDiscoveryConfig struct {
	CandidateLimit int
	DuplicateDays  int
	RecencyDays    int

	MatchWeights     RecommendationTypeScoreWeights
	ChallengeWeights RecommendationTypeScoreWeights

	ArtistRecency       float64
	GenreRecency        float64
	MinimumMatchScore   float64
	MinimumContactScore float64
	AlgorithmVersion    string
}

func DailyDiscoveryV1Config(base RecommendationConfig) DailyDiscoveryConfig {
	return DailyDiscoveryConfig{
		CandidateLimit:      base.CandidateLimit,
		DuplicateDays:       base.DuplicateDays,
		RecencyDays:         base.RecencyDays,
		MatchWeights:        RecommendationTypeScoreWeights{Mood: 2.5, Element: 2, Atmosphere: 2, Genre: 3, Artist: 1.5, Song: 1, Discovery: .25},
		ChallengeWeights:    RecommendationTypeScoreWeights{Mood: 1, Element: 1.5, Atmosphere: 1.5, Genre: .5, Artist: .5, Song: .5, Discovery: 4.5},
		ArtistRecency:       base.ScoreWeights.ArtistRecency,
		GenreRecency:        base.ScoreWeights.GenreRecency,
		MinimumMatchScore:   DailyDiscoveryMinimumMatchScore,
		MinimumContactScore: DailyDiscoveryMinimumContactScore,
		AlgorithmVersion:    "daily-discovery-v1",
	}
}
