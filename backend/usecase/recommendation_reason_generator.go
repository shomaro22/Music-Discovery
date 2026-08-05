package usecase

import (
	"fmt"
	"sort"

	"music-discovery/backend/entity"
)

type recommendationReasonPart struct {
	name  string
	score float64
}

func recommendationReason(kind entity.RecommendationType, candidate SongCandidate) string {
	parts := []recommendationReasonPart{{"気分", candidate.Score.Mood}, {"重視する要素", candidate.Score.Element}, {"雰囲気", candidate.Score.Atmosphere}, {"ジャンル", candidate.Score.Genre}, {"新しさ", candidate.Score.Novelty}, {"探索性", candidate.Score.Discovery}}
	sort.SliceStable(parts, func(i, j int) bool { return parts[i].score > parts[j].score })
	names := make([]string, 0, 2)
	for _, part := range parts {
		if part.score <= 0 {
			continue
		}
		names = append(names, part.name)
		if len(names) == 2 {
			break
		}
	}
	if len(names) == 0 {
		switch kind {
		case entity.RecommendationTypeYourMatch:
			return "今の気分と好みに近い1曲です"
		case entity.RecommendationTypeNewFavorite:
			return "好みに近い、まだ出会う機会が少ない1曲です"
		default:
			return "好みとの接点を残しながら、新しい方向性を楽しめる1曲です"
		}
	}
	joined := names[0]
	if len(names) == 2 {
		joined = names[0] + "と" + names[1]
	}
	switch kind {
	case entity.RecommendationTypeYourMatch:
		return fmt.Sprintf("%sとの一致度が高い1曲です", joined)
	case entity.RecommendationTypeNewFavorite:
		return fmt.Sprintf("%sに合う、まだ出会う機会が少ない1曲です", joined)
	default:
		return fmt.Sprintf("%sとの接点を残しながら、普段と異なる方向性を楽しめる1曲です", joined)
	}
}
