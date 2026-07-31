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
	candidateImport ISpotifyCandidateImportUsecase
	clock           IClock
	config          RecommendationConfig
}

func NewRecommendationUsecase(users repository.IUserRepository, questions repository.IQuestionRepository, answers repository.IQuestionAnswerRepository, profiles repository.IUserProfileRepository, recommendations repository.IRecommendationRepository, locks repository.IRecommendationLockRepository, tx repository.ITransactionManager, calculator IRecommendationCalculator, candidateImport ISpotifyCandidateImportUsecase, clock IClock, config RecommendationConfig) IRecommendationUsecase {
	return &recommendationUsecase{users: users, questions: questions, answers: answers, profiles: profiles, recommendations: recommendations, locks: locks, tx: tx, calculator: calculator, candidateImport: candidateImport, clock: clock, config: config}
}

func (u *recommendationUsecase) Generate(ctx context.Context, userID uuid.UUID) (*entity.Recommendation, error) {
	lockTTLSeconds := u.config.LockTTLSeconds
	if lockTTLSeconds <= 0 {
		lockTTLSeconds = 120
	}
	token, acquired, err := u.locks.Acquire(ctx, userID, time.Duration(lockTTLSeconds)*time.Second)
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
	if len(candidates) < 3 && u.candidateImport != nil {
		imported, importErr := u.candidateImport.Import(ctx, condition.Genres)
		if importErr != nil {
			return nil, importErr
		}
		if imported > 0 {
			// Spotify取込直後のSongには独自Tagを付与しない。管理者による
			// Tag確認が完了するまで推薦対象外として扱う。
			candidates, err = u.recommendations.FindCandidates(ctx, repository.RecommendationCandidateCondition{ExcludedSongIDs: excluded, Limit: u.config.CandidateLimit})
			if err != nil {
				return nil, entity.ErrRecommendationGenerateFailed
			}
			if len(candidates) < 3 {
				return nil, entity.ErrRecommendationCandidatesImportedPendingReview
			}
		}
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
	recency, err := u.recommendations.FindRecentDisplayCounts(ctx, userID, now.AddDate(0, 0, -u.config.RecencyDays))
	if err != nil {
		return nil, entity.ErrRecommendationGenerateFailed
	}
	condition.RecentArtistCounts = recency.ArtistCounts
	condition.RecentGenreCounts = recency.GenreCounts
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
		candidate, ok := u.pickCandidate(ranked, kind, condition, artistCounts, songCounts, selectedSongs, selectedArtists, first)
		if !ok {
			return nil, entity.ErrRecommendationCandidatesNotFound
		}
		selectedSongs[candidate.Data.Song.ID] = true
		selectedArtists[candidate.Data.Artist.ID] = true
		result = append(result, entity.RecommendationDetail{SongID: candidate.Data.Song.ID, Type: kind, Score: candidate.Score.Total, Position: position + 1, Reason: reasonFor(kind, candidate.Data)})
	}
	return result, nil
}

func (u *recommendationUsecase) pickCandidate(ranked []SongCandidate, kind entity.RecommendationType, condition RecommendationCondition, artistCounts map[uuid.UUID]int64, songCounts map[uuid.UUID]int64, selectedSongs map[uuid.UUID]bool, selectedArtists map[uuid.UUID]bool, first bool) (SongCandidate, bool) {
	// Strict -> relaxed -> fallback. Song duplication and the Try Something New
	// mood/element/atmosphere bridge are never relaxed.
	for stage := 0; stage < 3; stage++ {
		for _, candidate := range ranked {
			if selectedSongs[candidate.Data.Song.ID] {
				continue
			}
			contact := candidate.Score.Mood + candidate.Score.Element + candidate.Score.Atmosphere
			genreMatch := candidateMatchesGenres(candidate.Data, condition.Genres)
			likedArtist := condition.ArtistWeights[candidate.Data.Artist.ID] > 0
			seenArtist := artistCounts[candidate.Data.Artist.ID] > 0
			if kind == entity.RecommendationTypeTrySomethingNew {
				if selectedArtists[candidate.Data.Artist.ID] {
					continue
				}
				if contact <= 0 || !matchesDiscoveryStage(candidate.Data, condition, seenArtist, stage) {
					continue
				}
			}
			if first {
				if stage == 0 && selectedArtists[candidate.Data.Artist.ID] {
					continue
				}
				if kind != entity.RecommendationTypeTrySomethingNew && stage == 0 && candidate.Score.Mood+candidate.Score.Element+candidate.Score.Atmosphere+candidate.Score.Genre < u.config.MinimumMatchScore {
					continue
				}
				return candidate, true
			}
			seenSong := songCounts[candidate.Data.Song.ID] > 0
			switch kind {
			case entity.RecommendationTypeYourMatch:
				// 2回目以降は、回答Genreに一致し、履歴上未推薦のArtistを優先する。
				if stage == 0 && (!genreMatch || seenArtist || selectedArtists[candidate.Data.Artist.ID]) {
					continue
				}
				if stage == 1 && (!genreMatch || selectedArtists[candidate.Data.Artist.ID]) {
					continue
				}
			case entity.RecommendationTypeNewFavorite:
				// 高評価Artistのうち、過去に推薦していない別Songを最優先する。
				if stage == 0 && (!likedArtist || seenSong || selectedArtists[candidate.Data.Artist.ID]) {
					continue
				}
				if stage == 1 && (!likedArtist || seenSong) {
					continue
				}
			case entity.RecommendationTypeTrySomethingNew:
				// 履歴上も今回の3枠内でも、Genre・Artistが異なる候補を優先する。
				seenGenre := candidateHasRecommendedGenre(candidate.Data, condition.RecommendedGenres)
				if stage == 0 && (seenGenre || seenArtist || selectedArtists[candidate.Data.Artist.ID]) {
					continue
				}
				if stage == 1 && (seenArtist || selectedArtists[candidate.Data.Artist.ID]) {
					continue
				}
			}
			return candidate, true
		}
	}
	return SongCandidate{}, false
}

func candidateGenreKeys(value entity.RecommendationSongCandidate) []string {
	return songGenreKeys(value)
}

func candidateMatchesGenres(value entity.RecommendationSongCandidate, genres []string) bool {
	wanted := make(map[string]bool, len(genres))
	for _, genre := range genres {
		wanted[normalized(genre)] = true
	}
	for _, genre := range candidateGenreKeys(value) {
		if wanted[genre] {
			return true
		}
	}
	return false
}

func candidateHasRecommendedGenre(value entity.RecommendationSongCandidate, recommended map[string]bool) bool {
	for _, genre := range candidateGenreKeys(value) {
		if recommended[genre] {
			return true
		}
	}
	return false
}

var nearbyGenres = map[string]map[string]bool{
	// Chapter 10 genre-distance table. Every selectable genre is represented,
	// and all relationships are intentionally symmetric.
	"pop":         {"electronic": true, "k-pop": true, "anime": true, "hip-hop": true},
	"rock":        {"alternative": true, "metal": true},
	"alternative": {"rock": true, "metal": true},
	"metal":       {"rock": true, "alternative": true},
	"hip-hop":     {"pop": true, "k-pop": true, "electronic": true},
	"jazz":        {"classical": true},
	"electronic":  {"pop": true, "k-pop": true, "anime": true, "hip-hop": true},
	"classical":   {"jazz": true, "anime": true},
	"anime":       {"pop": true, "electronic": true, "k-pop": true, "classical": true},
	"k-pop":       {"pop": true, "electronic": true, "hip-hop": true, "anime": true},
}

func candidateIsNearbyGenre(value entity.RecommendationSongCandidate, genres []string) bool {
	for _, current := range genres {
		current = normalized(current)
		for _, candidate := range candidateGenreKeys(value) {
			if nearbyGenres[current][candidate] {
				return true
			}
		}
	}
	return false
}

func matchesDiscoveryStage(value entity.RecommendationSongCandidate, condition RecommendationCondition, seenArtist bool, stage int) bool {
	sameGenre := candidateMatchesGenres(value, condition.Genres)
	nearGenre := candidateIsNearbyGenre(value, condition.Genres)
	seenGenre := candidateHasRecommendedGenre(value, condition.RecommendedGenres)
	unknownArtist := !seenArtist

	switch effectiveDiscoveryLevel(condition.DiscoveryLevel, condition.DiscoveryAdjustment) {
	case "low":
		if stage == 0 {
			return sameGenre && unknownArtist
		}
		if stage == 1 {
			return (sameGenre || nearGenre) && unknownArtist
		}
		return sameGenre || nearGenre || unknownArtist
	case "high":
		if stage == 0 {
			return !seenGenre && !sameGenre && unknownArtist
		}
		if stage == 1 {
			return !sameGenre && unknownArtist
		}
		return !sameGenre || unknownArtist
	default: // medium
		if stage == 0 {
			return nearGenre && unknownArtist
		}
		if stage == 1 {
			return nearGenre || (!seenGenre && unknownArtist)
		}
		return !sameGenre || unknownArtist
	}
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
	condition := RecommendationCondition{GenreWeights: map[string]float64{}, ElementWeights: map[string]float64{}, AtmosphereWeights: map[string]float64{}, ArtistWeights: map[uuid.UUID]float64{}, SongWeights: map[uuid.UUID]float64{}}
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
	condition.DiscoveryAdjustment = profile.DiscoveryAdjustment
	allChoiceIDs := []uuid.UUID{}
	for _, v := range profile.GenreWeights {
		allChoiceIDs = append(allChoiceIDs, v.ChoiceID)
	}
	for _, v := range profile.ElementWeights {
		allChoiceIDs = append(allChoiceIDs, v.ChoiceID)
	}
	for _, v := range profile.AtmosphereWeights {
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
	for _, v := range profile.ElementWeights {
		condition.ElementWeights[codeByID[v.ChoiceID]] = v.Weight
	}
	for _, v := range profile.AtmosphereWeights {
		condition.AtmosphereWeights[codeByID[v.ChoiceID]] = v.Weight
	}
	condition.ArtistWeights, err = u.recommendations.FindArtistWeights(ctx, userID)
	if err != nil {
		return RecommendationCondition{}, entity.ErrRecommendationGenerateFailed
	}
	condition.SongWeights, err = u.recommendations.FindSongWeights(ctx, userID)
	if err != nil {
		return RecommendationCondition{}, entity.ErrRecommendationGenerateFailed
	}
	historyGenres, err := u.recommendations.FindRecommendedGenres(ctx, userID)
	if err != nil {
		return RecommendationCondition{}, entity.ErrRecommendationGenerateFailed
	}
	condition.RecommendedGenres = make(map[string]bool, len(historyGenres))
	for _, genre := range historyGenres {
		condition.RecommendedGenres[normalized(genre)] = true
	}
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
