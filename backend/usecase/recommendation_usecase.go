package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"music-discovery/backend/entity"
	"music-discovery/backend/repository"
)

type IRecommendationUsecase interface {
	Generate(context.Context, uuid.UUID) (*entity.Recommendation, error)
	GetLatest(context.Context, uuid.UUID) (*entity.Recommendation, error)
	GetByID(context.Context, uuid.UUID, uuid.UUID) (*entity.Recommendation, error)
}
type recommendationUsecase struct {
	users           repository.IUserRepository
	questions       repository.IQuestionRepository
	answers         repository.IQuestionAnswerRepository
	profiles        repository.IUserProfileRepository
	recommendations repository.IRecommendationRepository
	locks           repository.IRecommendationLockRepository
	tx              repository.ITransactionManager
	calculator      IRecommendationCalculator
	clock           IClock
	config          RecommendationConfig
}

func NewRecommendationUsecase(users repository.IUserRepository, questions repository.IQuestionRepository, answers repository.IQuestionAnswerRepository, profiles repository.IUserProfileRepository, recommendations repository.IRecommendationRepository, locks repository.IRecommendationLockRepository, tx repository.ITransactionManager, calculator IRecommendationCalculator, clock IClock, config RecommendationConfig) IRecommendationUsecase {
	return &recommendationUsecase{users: users, questions: questions, answers: answers, profiles: profiles, recommendations: recommendations, locks: locks, tx: tx, calculator: calculator, clock: clock, config: config}
}

func (u *recommendationUsecase) Generate(ctx context.Context, userID uuid.UUID) (*entity.Recommendation, error) {
	token, acquired, err := u.locks.Acquire(ctx, userID, time.Duration(u.config.GenerationWindow)*time.Second)
	if err != nil {
		return nil, entity.ErrRecommendationGenerateFailed
	}
	if !acquired {
		return nil, entity.ErrRecommendationAlreadyGenerated
	}
	defer func() { _ = u.locks.Release(context.Background(), userID, token) }()

	user, err := u.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.Status == entity.UserStatusSuspended {
		return nil, entity.ErrUserSuspended
	}
	now := u.clock.Now()
	recent, err := u.recommendations.HasRecentNormal(ctx, userID, now.Add(-time.Duration(u.config.GenerationWindow)*time.Second))
	if err != nil {
		return nil, entity.ErrRecommendationGenerateFailed
	}
	if recent {
		return nil, entity.ErrRecommendationAlreadyGenerated
	}

	condition, err := u.buildCondition(ctx, userID)
	if err != nil {
		return nil, err
	}
	excluded, err := u.recommendations.FindRecommendedSongIDs(ctx, userID, now.AddDate(0, 0, -u.config.DuplicateDays), now)
	if err != nil {
		return nil, entity.ErrRecommendationGenerateFailed
	}
	candidates, err := u.recommendations.FindCandidates(ctx, repository.RecommendationCandidateCondition{ExcludedSongIDs: excluded, Limit: u.config.CandidateLimit})
	if err != nil {
		return nil, entity.ErrRecommendationGenerateFailed
	}
	if len(candidates) < 3 {
		return nil, entity.ErrRecommendationCandidatesNotFound
	}
	artistCounts, err := u.recommendations.CountByArtist(ctx, userID)
	if err != nil {
		return nil, entity.ErrRecommendationGenerateFailed
	}
	songCounts, err := u.recommendations.CountBySong(ctx, userID)
	if err != nil {
		return nil, entity.ErrRecommendationGenerateFailed
	}
	firstRecommendation := len(artistCounts) == 0 && len(songCounts) == 0

	details, err := u.selectDetails(condition, candidates, artistCounts, songCounts, firstRecommendation)
	if err != nil {
		return nil, err
	}
	recommendation := &entity.Recommendation{UserID: userID, Source: entity.RecommendationSourceNormal, RecommendedAt: now, AlgorithmVersion: u.config.AlgorithmVersion, Details: details}
	if err := u.tx.WithinTransaction(ctx, func(tx repository.ITransaction) error { return u.recommendations.Create(ctx, tx, recommendation) }); err != nil {
		return nil, entity.ErrRecommendationSaveFailed
	}
	return u.recommendations.FindByID(ctx, recommendation.ID)
}

func (u *recommendationUsecase) selectDetails(condition RecommendationCondition, candidates []entity.RecommendationSongCandidate, artistCounts map[uuid.UUID]int64, songCounts map[uuid.UUID]int64, first bool) ([]entity.RecommendationDetail, error) {
	selectedSongs := map[uuid.UUID]bool{}
	selectedArtists := map[uuid.UUID]bool{}
	result := make([]entity.RecommendationDetail, 0, 3)
	kinds := []entity.RecommendationType{entity.RecommendationTypeYourMatch, entity.RecommendationTypeNewFavorite, entity.RecommendationTypeTrySomethingNew}
	for position, kind := range kinds {
		ranked := u.calculator.Calculate(condition, candidates, kind, artistCounts, songCounts)
		candidate, ok := u.pickCandidate(ranked, kind, condition, selectedSongs, selectedArtists, first)
		if !ok {
			return nil, entity.ErrRecommendationCandidatesNotFound
		}
		selectedSongs[candidate.Data.Song.ID] = true
		selectedArtists[candidate.Data.Artist.ID] = true
		result = append(result, entity.RecommendationDetail{SongID: candidate.Data.Song.ID, Type: kind, Score: candidate.Score.Total, Position: position + 1, Reason: reasonFor(kind, candidate.Data)})
	}
	return result, nil
}

func (u *recommendationUsecase) pickCandidate(ranked []SongCandidate, kind entity.RecommendationType, condition RecommendationCondition, selectedSongs map[uuid.UUID]bool, selectedArtists map[uuid.UUID]bool, first bool) (SongCandidate, bool) {
	// Strict -> relaxed -> fallback. The 30-day and session song exclusions are never relaxed.
	for stage := 0; stage < 3; stage++ {
		for _, candidate := range ranked {
			if selectedSongs[candidate.Data.Song.ID] {
				continue
			}
			contact := candidate.Score.Mood + candidate.Score.Element + candidate.Score.Atmosphere
			genreMatch := candidate.Score.Genre > 0
			knownArtist := condition.ArtistWeights[candidate.Data.Artist.ID] > 0
			if kind == entity.RecommendationTypeTrySomethingNew && contact <= 0 {
				continue
			}
			if candidate.Score.FeedbackPenalty >= 3 && stage < 2 {
				continue
			}
			if first {
				if stage == 0 && selectedArtists[candidate.Data.Artist.ID] {
					continue
				}
				if kind != entity.RecommendationTypeTrySomethingNew && stage == 0 && candidate.Score.Mood+candidate.Score.Element+candidate.Score.Atmosphere+candidate.Score.Genre < u.config.MinimumMatchScore {
					continue
				}
				if kind == entity.RecommendationTypeTrySomethingNew && stage == 0 && genreMatch && knownArtist {
					continue
				}
				return candidate, true
			}
			switch kind {
			case entity.RecommendationTypeYourMatch:
				// Second and later: same/related genre, but a different artist where possible.
				if stage == 0 && (!genreMatch || selectedArtists[candidate.Data.Artist.ID]) {
					continue
				}
				if stage == 1 && !genreMatch {
					continue
				}
			case entity.RecommendationTypeNewFavorite:
				// Second and later: a different song from a positively learned artist.
				if stage == 0 && !knownArtist {
					continue
				}
				if stage == 1 && !genreMatch {
					continue
				}
			case entity.RecommendationTypeTrySomethingNew:
				// Keep a mood/element/atmosphere bridge while preferring both a new genre and artist.
				if stage == 0 && (genreMatch || knownArtist) {
					continue
				}
				if stage == 1 && knownArtist {
					continue
				}
			}
			return candidate, true
		}
	}
	return SongCandidate{}, false
}

func reasonFor(kind entity.RecommendationType, value entity.RecommendationSongCandidate) string {
	switch kind {
	case entity.RecommendationTypeYourMatch:
		return fmt.Sprintf("今の気分と好みに近い%sの楽曲です", value.Artist.Name)
	case entity.RecommendationTypeNewFavorite:
		return "好みに近く、まだ出会う機会が少ない楽曲です"
	default:
		return "好みとの接点を残しながら、新しい音楽体験を広げる1曲です"
	}
}

func (u *recommendationUsecase) buildCondition(ctx context.Context, userID uuid.UUID) (RecommendationCondition, error) {
	session, err := u.answers.GetLatest(ctx, userID)
	if errors.Is(err, entity.ErrNotFound) {
		return RecommendationCondition{}, entity.ErrQuestionAnswerRequired
	}
	if err != nil {
		return RecommendationCondition{}, entity.ErrRecommendationGenerateFailed
	}
	questions, err := u.questions.ListActive(ctx)
	if err != nil {
		return RecommendationCondition{}, entity.ErrRecommendationGenerateFailed
	}
	questionCode := map[uuid.UUID]string{}
	for _, q := range questions {
		questionCode[q.ID] = q.Code
	}
	ids := []uuid.UUID{}
	answerQuestion := map[uuid.UUID]string{}
	for _, answer := range session.Answers {
		for _, choice := range answer.Choices {
			ids = append(ids, choice.QuestionChoiceID)
			answerQuestion[choice.QuestionChoiceID] = questionCode[answer.QuestionID]
		}
	}
	choices, err := u.questions.FindChoicesByIDs(ctx, ids)
	if err != nil {
		return RecommendationCondition{}, entity.ErrRecommendationGenerateFailed
	}
	condition := RecommendationCondition{GenreWeights: map[string]float64{}, ArtistWeights: map[uuid.UUID]float64{}, SongWeights: map[uuid.UUID]float64{}, ArtistFeedbackPenalties: map[uuid.UUID]float64{}, SongFeedbackPenalties: map[uuid.UUID]float64{}}
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
	err = u.tx.WithinTransaction(ctx, func(tx repository.ITransaction) error {
		var findErr error
		profile, findErr = u.profiles.FindByUserID(ctx, tx, userID)
		return findErr
	})
	if errors.Is(err, entity.ErrNotFound) {
		return RecommendationCondition{}, entity.ErrUserProfileRequired
	}
	if err != nil {
		return RecommendationCondition{}, entity.ErrRecommendationGenerateFailed
	}
	condition.DiscoveryLevel = profile.DiscoveryLevel
	allChoiceIDs := []uuid.UUID{}
	for _, v := range profile.GenreWeights {
		allChoiceIDs = append(allChoiceIDs, v.ChoiceID)
	}
	weightChoices, err := u.questions.FindChoicesByIDs(ctx, allChoiceIDs)
	if err != nil {
		return RecommendationCondition{}, entity.ErrRecommendationGenerateFailed
	}
	codeByID := map[uuid.UUID]string{}
	for _, v := range weightChoices {
		codeByID[v.ID] = normalized(v.Code)
	}
	for _, v := range profile.GenreWeights {
		condition.GenreWeights[codeByID[v.ChoiceID]] = v.Weight
	}
	signals, err := u.recommendations.FindFeedbackSignals(ctx, userID)
	if err != nil {
		return RecommendationCondition{}, entity.ErrRecommendationGenerateFailed
	}
	condition.ArtistWeights = signals.ArtistWeights
	condition.SongWeights = signals.SongWeights
	condition.ArtistFeedbackPenalties = signals.ArtistPenalties
	condition.SongFeedbackPenalties = signals.SongPenalties
	return condition, nil
}
func (u *recommendationUsecase) GetLatest(ctx context.Context, userID uuid.UUID) (*entity.Recommendation, error) {
	return u.recommendations.FindLatest(ctx, userID, entity.RecommendationSourceNormal)
}
func (u *recommendationUsecase) GetByID(ctx context.Context, userID, id uuid.UUID) (*entity.Recommendation, error) {
	value, err := u.recommendations.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if value.UserID != userID {
		return nil, entity.ErrRecommendationAccessDenied
	}
	return value, nil
}
