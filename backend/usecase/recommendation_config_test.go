package usecase

import "testing"

func TestDefaultRecommendationScoreWeightsMatchSpecification(t *testing.T) {
	weights := DefaultRecommendationScoreWeights()

	tests := []struct {
		name string
		got  float64
		want float64
	}{
		{name: "your match feedback penalty", got: weights.YourMatch.FeedbackPenalty, want: 1.5},
		{name: "new favorite element", got: weights.NewFavorite.Element, want: 1.5},
		{name: "new favorite atmosphere", got: weights.NewFavorite.Atmosphere, want: 1.5},
		{name: "new favorite novelty", got: weights.NewFavorite.Novelty, want: 2.5},
		{name: "new favorite feedback penalty", got: weights.NewFavorite.FeedbackPenalty, want: 1.5},
		{name: "new favorite recency penalty", got: weights.NewFavorite.RecencyPenalty, want: 1.5},
		{name: "try something new novelty", got: weights.TrySomethingNew.Novelty, want: 3},
		{name: "try something new feedback penalty", got: weights.TrySomethingNew.FeedbackPenalty, want: 1.5},
		{name: "try something new recency penalty", got: weights.TrySomethingNew.RecencyPenalty, want: 1.5},
		{name: "daily match feedback penalty", got: weights.DailyMatch.FeedbackPenalty, want: 1.5},
		{name: "daily challenge element", got: weights.DailyChallenge.Element, want: 1.5},
		{name: "daily challenge atmosphere", got: weights.DailyChallenge.Atmosphere, want: 1.5},
		{name: "daily challenge discovery", got: weights.DailyChallenge.Discovery, want: 4.5},
		{name: "daily challenge novelty", got: weights.DailyChallenge.Novelty, want: 3},
		{name: "daily challenge feedback penalty", got: weights.DailyChallenge.FeedbackPenalty, want: 1.5},
		{name: "daily challenge recency penalty", got: weights.DailyChallenge.RecencyPenalty, want: 1.5},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Fatalf("got=%v want=%v", test.got, test.want)
			}
		})
	}
}

func TestHasMinimumContactUsesConfiguredThreshold(t *testing.T) {
	candidate := SongCandidate{Score: RecommendationScore{Mood: 1}}
	condition := RecommendationCondition{}

	if !hasMinimumContact(candidate, condition, 1) {
		t.Fatal("one contact should satisfy threshold 1")
	}
	if hasMinimumContact(candidate, condition, 2) {
		t.Fatal("one contact must not satisfy threshold 2")
	}
}
