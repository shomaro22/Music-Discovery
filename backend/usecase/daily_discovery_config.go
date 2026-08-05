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
		MatchWeights:        base.ScoreWeights.DailyMatch,
		ChallengeWeights:    base.ScoreWeights.DailyChallenge,
		ArtistRecency:       base.ScoreWeights.ArtistRecency,
		GenreRecency:        base.ScoreWeights.GenreRecency,
		MinimumMatchScore:   DailyDiscoveryMinimumMatchScore,
		MinimumContactScore: DailyDiscoveryMinimumContactScore,
		AlgorithmVersion:    "daily-discovery-v1",
	}
}
