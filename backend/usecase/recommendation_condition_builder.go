package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"music-discovery/backend/entity"
	"music-discovery/backend/repository"
)

type IRecommendationConditionBuilder interface {
	Build(context.Context, uuid.UUID) (RecommendationCondition, error)
}

type recommendationConditionBuilder struct {
	questions       repository.IQuestionRepository
	answers         repository.IQuestionAnswerRepository
	profiles        repository.IUserProfileRepository
	recommendations repository.IRecommendationRepository
	tx              repository.ITransactionManager
}

func NewRecommendationConditionBuilder(
	questions repository.IQuestionRepository,
	answers repository.IQuestionAnswerRepository,
	profiles repository.IUserProfileRepository,
	recommendations repository.IRecommendationRepository,
	tx repository.ITransactionManager,
) IRecommendationConditionBuilder {
	return &recommendationConditionBuilder{
		questions: questions, answers: answers, profiles: profiles,
		recommendations: recommendations, tx: tx,
	}
}

func (b *recommendationConditionBuilder) Build(ctx context.Context, userID uuid.UUID) (RecommendationCondition, error) {
	session, err := b.answers.GetLatest(ctx, userID)
	if errors.Is(err, entity.ErrNotFound) {
		return RecommendationCondition{}, entity.ErrQuestionAnswerRequired
	}
	if err != nil {
		return RecommendationCondition{}, entity.ErrRecommendationGenerateFailed
	}
	questions, err := b.questions.ListActive(ctx)
	if err != nil {
		return RecommendationCondition{}, entity.ErrRecommendationGenerateFailed
	}
	questionCode := map[uuid.UUID]string{}
	for _, question := range questions {
		questionCode[question.ID] = question.Code
	}
	ids := []uuid.UUID{}
	answerQuestion := map[uuid.UUID]string{}
	for _, answer := range session.Answers {
		for _, choice := range answer.Choices {
			ids = append(ids, choice.QuestionChoiceID)
			answerQuestion[choice.QuestionChoiceID] = questionCode[answer.QuestionID]
		}
	}
	choices, err := b.questions.FindChoicesByIDs(ctx, ids)
	if err != nil {
		return RecommendationCondition{}, entity.ErrRecommendationGenerateFailed
	}
	condition := RecommendationCondition{
		GenreWeights: map[string]float64{}, ElementWeights: map[string]float64{},
		AtmosphereWeights: map[string]float64{}, ArtistWeights: map[uuid.UUID]float64{},
		SongWeights: map[uuid.UUID]float64{},
	}
	for _, choice := range choices {
		switch answerQuestion[choice.ID] {
		case entity.QuestionCodeMood:
			condition.Mood = append(condition.Mood, choice.Code)
		case entity.QuestionCodeElement:
			condition.Elements = append(condition.Elements, choice.Code)
		case entity.QuestionCodeAtmosphere:
			condition.Atmospheres = append(condition.Atmospheres, choice.Code)
		case entity.QuestionCodeGenre:
			condition.Genres = append(condition.Genres, choice.Code)
		case entity.QuestionCodeDiscoveryLevel:
			condition.DiscoveryLevel = choice.Code
		}
	}
	if len(condition.Mood) == 0 || len(condition.Elements) == 0 || len(condition.Atmospheres) == 0 || len(condition.Genres) == 0 || strings.TrimSpace(condition.DiscoveryLevel) == "" {
		return RecommendationCondition{}, entity.ErrQuestionAnswerRequired
	}
	var profile *entity.UserProfile
	err = b.tx.WithinTransaction(ctx, func(tx repository.ITransaction) error {
		var findErr error
		profile, findErr = b.profiles.FindByUserID(ctx, tx, userID)
		return findErr
	})
	if errors.Is(err, entity.ErrNotFound) {
		return RecommendationCondition{}, entity.ErrUserProfileRequired
	}
	if err != nil {
		return RecommendationCondition{}, entity.ErrRecommendationGenerateFailed
	}
	condition.DiscoveryLevel = profile.DiscoveryLevel
	condition.DiscoveryAdjustment = profile.DiscoveryAdjustment
	allChoiceIDs := []uuid.UUID{}
	for _, value := range profile.GenreWeights {
		allChoiceIDs = append(allChoiceIDs, value.ChoiceID)
	}
	for _, value := range profile.ElementWeights {
		allChoiceIDs = append(allChoiceIDs, value.ChoiceID)
	}
	for _, value := range profile.AtmosphereWeights {
		allChoiceIDs = append(allChoiceIDs, value.ChoiceID)
	}
	weightChoices, err := b.questions.FindChoicesByIDs(ctx, allChoiceIDs)
	if err != nil {
		return RecommendationCondition{}, entity.ErrRecommendationGenerateFailed
	}
	codeByID := map[uuid.UUID]string{}
	for _, value := range weightChoices {
		codeByID[value.ID] = normalized(value.Code)
	}
	for _, value := range profile.GenreWeights {
		condition.GenreWeights[codeByID[value.ChoiceID]] = value.Weight
	}
	for _, value := range profile.ElementWeights {
		condition.ElementWeights[codeByID[value.ChoiceID]] = value.Weight
	}
	for _, value := range profile.AtmosphereWeights {
		condition.AtmosphereWeights[codeByID[value.ChoiceID]] = value.Weight
	}
	condition.ArtistWeights, err = b.recommendations.FindArtistWeights(ctx, userID)
	if err != nil {
		return RecommendationCondition{}, entity.ErrRecommendationGenerateFailed
	}
	condition.SongWeights, err = b.recommendations.FindSongWeights(ctx, userID)
	if err != nil {
		return RecommendationCondition{}, entity.ErrRecommendationGenerateFailed
	}
	historyGenres, err := b.recommendations.FindRecommendedGenres(ctx, userID)
	if err != nil {
		return RecommendationCondition{}, entity.ErrRecommendationGenerateFailed
	}
	condition.RecommendedGenres = make(map[string]bool, len(historyGenres))
	for _, genre := range historyGenres {
		condition.RecommendedGenres[normalized(genre)] = true
	}
	return condition, nil
}
