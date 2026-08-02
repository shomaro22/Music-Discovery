package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"music-discovery/backend/entity"
	"music-discovery/backend/repository"
)

type DailyDiscoveryList struct {
	Items                   []entity.Recommendation
	Page, Limit, TotalPages int
	Total                   int64
}
type IDailyDiscoveryUsecase interface {
	GetToday(context.Context, uuid.UUID) (*entity.Recommendation, bool, error)
	List(context.Context, uuid.UUID, int, int) (*DailyDiscoveryList, error)
	GetByID(context.Context, uuid.UUID, uuid.UUID) (*entity.Recommendation, error)
}
type dailyDiscoveryUsecase struct{ recommendation *recommendationUsecase }

func NewDailyDiscoveryUsecase(users repository.IUserRepository, questions repository.IQuestionRepository, answers repository.IQuestionAnswerRepository, profiles repository.IUserProfileRepository, recommendations repository.IRecommendationRepository, tx repository.ITransactionManager, calculator IRecommendationCalculator, clock IClock, config RecommendationConfig) IDailyDiscoveryUsecase {
	return &dailyDiscoveryUsecase{recommendation: &recommendationUsecase{users: users, questions: questions, answers: answers, profiles: profiles, recommendations: recommendations, tx: tx, calculator: calculator, clock: clock, config: config}}
}
func (u *dailyDiscoveryUsecase) targetDate(ctx context.Context, userID uuid.UUID) (time.Time, error) {
	var profile *entity.UserProfile
	err := u.recommendation.tx.WithinTransaction(ctx, func(tx repository.ITransaction) error {
		var e error
		profile, e = u.recommendation.profiles.FindByUserID(ctx, tx, userID)
		return e
	})
	if errors.Is(err, entity.ErrNotFound) {
		return time.Time{}, entity.ErrUserProfileRequired
	}
	if err != nil {
		return time.Time{}, entity.ErrDailyDiscoveryFetchFailed
	}
	zone := profile.Timezone
	if zone == "" {
		zone = "Asia/Tokyo"
	}
	loc, e := time.LoadLocation(zone)
	if e != nil {
		loc, _ = time.LoadLocation("Asia/Tokyo")
	}
	now := u.recommendation.clock.Now().In(loc)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC), nil
}
func (u *dailyDiscoveryUsecase) GetToday(ctx context.Context, userID uuid.UUID) (*entity.Recommendation, bool, error) {
	user, err := u.recommendation.users.FindByID(ctx, userID)
	if err != nil {
		return nil, false, err
	}
	if user.Status == entity.UserStatusSuspended {
		return nil, false, entity.ErrUserSuspended
	}
	date, err := u.targetDate(ctx, userID)
	if err != nil {
		return nil, false, err
	}
	existing, err := u.recommendation.recommendations.FindDailyByDate(ctx, userID, date)
	if err == nil {
		return existing, false, nil
	}
	if !errors.Is(err, entity.ErrDailyDiscoveryNotFound) {
		return nil, false, entity.ErrDailyDiscoveryFetchFailed
	}
	condition, err := u.recommendation.buildCondition(ctx, userID)
	if err != nil {
		return nil, false, err
	}
	now := u.recommendation.clock.Now()
	excluded, err := u.recommendation.recommendations.FindRecommendedSongIDs(ctx, userID, now.AddDate(0, 0, -u.recommendation.config.DuplicateDays), now)
	if err != nil {
		return nil, false, entity.ErrDailyDiscoveryGenerateFailed
	}
	candidates, err := u.recommendation.recommendations.FindCandidates(ctx, repository.RecommendationCandidateCondition{ExcludedSongIDs: excluded, Limit: u.recommendation.config.CandidateLimit})
	if err != nil {
		return nil, false, entity.ErrDailyDiscoveryGenerateFailed
	}
	if len(candidates) < 3 {
		return nil, false, entity.ErrDailyDiscoveryCandidatesNotFound
	}
	artistCounts, _ := u.recommendation.recommendations.CountByArtist(ctx, userID)
	songCounts, _ := u.recommendation.recommendations.CountBySong(ctx, userID)
	recency, _ := u.recommendation.recommendations.FindRecentDisplayCounts(ctx, userID, now.AddDate(0, 0, -u.recommendation.config.RecencyDays))
	condition.RecentArtistCounts = recency.ArtistCounts
	condition.RecentGenreCounts = recency.GenreCounts
	match := u.recommendation.calculator.Calculate(condition, candidates, entity.RecommendationTypeYourMatch, artistCounts, songCounts)
	challenge := u.recommendation.calculator.Calculate(condition, candidates, entity.RecommendationTypeTrySomethingNew, artistCounts, songCounts)
	if len(match) < 2 || len(challenge) < 1 {
		return nil, false, entity.ErrDailyDiscoveryCandidatesNotFound
	}
	first := match[0]
	secondIndex := 1
	for i := 1; i < len(match); i++ {
		if match[i].Data.Artist.ID != first.Data.Artist.ID {
			secondIndex = i
			break
		}
	}
	second := match[secondIndex]
	var third SongCandidate
	found := false
	for _, c := range challenge {
		if c.Data.Song.ID != first.Data.Song.ID && c.Data.Song.ID != second.Data.Song.ID {
			third = c
			found = true
			break
		}
	}
	if !found {
		return nil, false, entity.ErrDailyDiscoveryCandidatesNotFound
	}
	d := date
	value := &entity.Recommendation{UserID: userID, Source: entity.RecommendationSourceDailyDiscovery, TargetDate: &d, RecommendedAt: now, AlgorithmVersion: u.recommendation.config.AlgorithmVersion, Details: []entity.RecommendationDetail{
		{SongID: first.Data.Song.ID, Type: entity.RecommendationTypeDailyMatchOne, Score: first.Score.Total, Position: 1, Reason: "あなたのジャンルや雰囲気に近い今日の1曲です"},
		{SongID: second.Data.Song.ID, Type: entity.RecommendationTypeDailyMatchTwo, Score: second.Score.Total, Position: 2, Reason: "重視する要素とアーティスト傾向に合う1曲です"},
		{SongID: third.Data.Song.ID, Type: entity.RecommendationTypeDailyChallenge, Score: third.Score.Total, Position: 3, Reason: "好みとの接点を残しながら、新しい要素を楽しめる1曲です"}}}
	err = u.recommendation.tx.WithinTransaction(ctx, func(tx repository.ITransaction) error { return u.recommendation.recommendations.Create(ctx, tx, value) })
	if err != nil {
		if saved, e := u.recommendation.recommendations.FindDailyByDate(ctx, userID, date); e == nil {
			return saved, false, nil
		}
		return nil, false, entity.ErrDailyDiscoverySaveFailed
	}
	saved, err := u.recommendation.recommendations.FindDailyByDate(ctx, userID, date)
	if err != nil {
		return nil, false, entity.ErrDailyDiscoveryFetchFailed
	}
	return saved, true, nil
}
func (u *dailyDiscoveryUsecase) List(ctx context.Context, userID uuid.UUID, page, limit int) (*DailyDiscoveryList, error) {
	if page < 1 || limit < 1 || limit > 100 {
		return nil, entity.ErrInvalidInput
	}
	items, total, err := u.recommendation.recommendations.ListDaily(ctx, userID, page, limit)
	if err != nil {
		return nil, err
	}
	pages := int((total + int64(limit) - 1) / int64(limit))
	return &DailyDiscoveryList{Items: items, Page: page, Limit: limit, Total: total, TotalPages: pages}, nil
}
func (u *dailyDiscoveryUsecase) GetByID(ctx context.Context, userID, id uuid.UUID) (*entity.Recommendation, error) {
	v, err := u.recommendation.recommendations.FindByID(ctx, id)
	if err != nil {
		return nil, entity.ErrDailyDiscoveryNotFound
	}
	if v.Source != entity.RecommendationSourceDailyDiscovery {
		return nil, entity.ErrDailyDiscoveryNotFound
	}
	if v.UserID != userID {
		return nil, entity.ErrDailyDiscoveryAccessDenied
	}
	return v, nil
}
