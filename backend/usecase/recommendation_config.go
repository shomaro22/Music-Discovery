package usecase

const (
	RecommendationAlgorithmVersion    = "recommendation-v2"
	RecommendationWeightConfigVersion = "weights-v2"
	RecommendationMinimumMatchScore   = 2
)

// DefaultRecommendationScoreWeights returns the version-controlled default score weights.
// These are algorithm settings, not environment-specific operational settings.
func DefaultRecommendationScoreWeights() RecommendationScoreWeights {
	return RecommendationScoreWeights{
		YourMatch:       RecommendationTypeScoreWeights{Mood: 3, Element: 1.5, Atmosphere: 1.5, Genre: 3, Artist: 1, Song: 1, Discovery: .25, Novelty: .25, FeedbackPenalty: 1, RecencyPenalty: 1},
		NewFavorite:     RecommendationTypeScoreWeights{Mood: 1.5, Element: 1, Atmosphere: 1, Genre: 2, Artist: 1, Song: 1, Discovery: 2, Novelty: 2, FeedbackPenalty: 1, RecencyPenalty: 1},
		TrySomethingNew: RecommendationTypeScoreWeights{Mood: 1, Element: 1, Atmosphere: 1, Genre: .5, Artist: .5, Song: .5, Discovery: 4, Novelty: 2, FeedbackPenalty: 1, RecencyPenalty: 1},
		DailyMatch:      RecommendationTypeScoreWeights{Mood: 2.5, Element: 2, Atmosphere: 2, Genre: 3, Artist: 1.5, Song: 1, Discovery: .25, Novelty: 1, FeedbackPenalty: 1, RecencyPenalty: 1},
		DailyChallenge:  RecommendationTypeScoreWeights{Mood: 1, Element: 1, Atmosphere: 1, Genre: .5, Artist: .5, Song: .5, Discovery: 4, Novelty: 2, FeedbackPenalty: 1, RecencyPenalty: 1},
		ArtistRecency:   .15,
		GenreRecency:    .3,
	}
}
