package usecase

const (
	RecommendationAlgorithmVersion  = "recommendation-v1"
	RecommendationMinimumMatchScore = 2
)

// RecommendationV1ScoreWeights returns the score weights that define
// recommendation-v1. These values are version-controlled algorithm settings,
// not environment-specific operational settings.
func RecommendationV1ScoreWeights() RecommendationScoreWeights {
	return RecommendationScoreWeights{
		YourMatch: RecommendationTypeScoreWeights{
			Mood: 3, Element: 1.5, Atmosphere: 1.5, Genre: 3,
			Artist: 1, Song: 1, Discovery: .25,
		},
		NewFavorite: RecommendationTypeScoreWeights{
			Mood: 1.5, Element: 1, Atmosphere: 1, Genre: 2,
			Artist: 1, Song: 1, Discovery: 2,
		},
		TrySomethingNew: RecommendationTypeScoreWeights{
			Mood: 1, Element: 1, Atmosphere: 1, Genre: .5,
			Artist: .5, Song: .5, Discovery: 4,
		},
		ArtistRecency: .15,
		GenreRecency:  .3,
	}
}
