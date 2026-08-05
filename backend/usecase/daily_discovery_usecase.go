package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
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

type dailyDiscoveryUsecase struct {
	users            repository.IUserRepository
	profiles         repository.IUserProfileRepository
	recommendations  repository.IRecommendationRepository
	locks            repository.IRecommendationLockRepository
	tx               repository.ITransactionManager
	calculator       IRecommendationCalculator
	clock            IClock
	conditionBuilder IRecommendationConditionBuilder
	config           DailyDiscoveryConfig
}

func NewDailyDiscoveryUsecase(
	users repository.IUserRepository,
	profiles repository.IUserProfileRepository,
	recommendations repository.IRecommendationRepository,
	locks repository.IRecommendationLockRepository,
	tx repository.ITransactionManager,
	calculator IRecommendationCalculator,
	clock IClock,
	conditionBuilder IRecommendationConditionBuilder,
	dailyConfig DailyDiscoveryConfig,
) IDailyDiscoveryUsecase {
	return &dailyDiscoveryUsecase{
		users:            users,
		profiles:         profiles,
		recommendations:  recommendations,
		locks:            locks,
		tx:               tx,
		calculator:       calculator,
		clock:            clock,
		conditionBuilder: conditionBuilder,
		config:           dailyConfig,
	}
}

func (u *dailyDiscoveryUsecase) targetDate(ctx context.Context, userID uuid.UUID) (time.Time, error) {
	var profile *entity.UserProfile
	err := u.tx.WithinTransaction(ctx, func(tx repository.ITransaction) error {
		var findErr error
		profile, findErr = u.profiles.FindByUserID(ctx, tx, userID)
		return findErr
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

	location, loadErr := time.LoadLocation(zone)
	if loadErr != nil || location == nil {
		// AlpineなどにTimezoneデータがない環境でもpanicさせない。
		// 初期TimezoneのAsia/Tokyoを固定オフセットで安全に使用する。
		location = time.FixedZone("Asia/Tokyo", 9*60*60)
	}

	now := u.clock.Now().In(location)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC), nil
}

func (u *dailyDiscoveryUsecase) GetToday(ctx context.Context, userID uuid.UUID) (*entity.Recommendation, bool, error) {
	user, err := u.users.FindByID(ctx, userID)
	if errors.Is(err, entity.ErrNotFound) || errors.Is(err, entity.ErrUserNotFound) {
		return nil, false, entity.ErrUserNotFound
	}
	if err != nil {
		return nil, false, entity.ErrDailyDiscoveryFetchFailed
	}
	if user.Status == entity.UserStatusSuspended {
		return nil, false, entity.ErrUserSuspended
	}

	targetDate, err := u.targetDate(ctx, userID)
	if err != nil {
		return nil, false, err
	}
	if existing, findErr := u.recommendations.FindDailyByDate(ctx, userID, targetDate); findErr == nil {
		return existing, false, nil
	} else if !errors.Is(findErr, entity.ErrDailyDiscoveryNotFound) {
		return nil, false, entity.ErrDailyDiscoveryFetchFailed
	}

	lockTTLSeconds := u.config.LockTTLSeconds
	if lockTTLSeconds <= 0 {
		lockTTLSeconds = 120
	}
	token, acquired, lockErr := u.locks.Acquire(
		ctx,
		userID,
		time.Duration(lockTTLSeconds)*time.Second,
	)
	if lockErr != nil {
		return nil, false, entity.ErrDailyDiscoveryGenerateFailed
	}
	if !acquired {
		return nil, false, entity.ErrDailyDiscoveryGenerationConflict
	}
	defer func() { _ = u.locks.Release(context.Background(), userID, token) }()
	if existing, findErr := u.recommendations.FindDailyByDate(ctx, userID, targetDate); findErr == nil {
		return existing, false, nil
	} else if !errors.Is(findErr, entity.ErrDailyDiscoveryNotFound) {
		return nil, false, entity.ErrDailyDiscoveryFetchFailed
	}

	condition, err := u.conditionBuilder.Build(ctx, userID)
	if err != nil {
		if errors.Is(err, entity.ErrQuestionAnswerRequired) || errors.Is(err, entity.ErrUserProfileRequired) {
			return nil, false, err
		}
		return nil, false, entity.ErrDailyDiscoveryGenerateFailed
	}
	now := u.clock.Now()
	excluded, err := u.recommendations.FindRecommendedSongIDs(
		ctx,
		userID,
		now.AddDate(0, 0, -u.config.DuplicateDays),
		now,
	)
	if err != nil {
		return nil, false, entity.ErrDailyDiscoveryGenerateFailed
	}
	condition.ExcludedSongIDs = make(map[uuid.UUID]bool, len(excluded))
	for _, songID := range excluded {
		condition.ExcludedSongIDs[songID] = true
	}
	candidates, err := u.recommendations.FindCandidates(
		ctx,
		repository.RecommendationCandidateCondition{
			ExcludedSongIDs: excluded,
			Limit:           u.config.CandidateLimit,
		},
	)
	if err != nil {
		return nil, false, entity.ErrDailyDiscoveryGenerateFailed
	}
	if len(candidates) < 3 {
		return nil, false, entity.ErrDailyDiscoveryCandidatesNotFound
	}

	artistCounts, err := u.recommendations.CountByArtist(ctx, userID)
	if err != nil {
		return nil, false, entity.ErrDailyDiscoveryGenerateFailed
	}
	songCounts, err := u.recommendations.CountBySong(ctx, userID)
	if err != nil {
		return nil, false, entity.ErrDailyDiscoveryGenerateFailed
	}
	recency, err := u.recommendations.FindRecentDisplayCounts(
		ctx,
		userID,
		now.AddDate(0, 0, -u.config.RecencyDays),
	)
	if err != nil {
		return nil, false, entity.ErrDailyDiscoveryGenerateFailed
	}
	condition.RecentArtistCounts = recency.ArtistCounts
	condition.RecentGenreCounts = recency.GenreCounts

	matchRanked, err := u.calculator.Calculate(condition, candidates, entity.RecommendationTypeDailyMatchOne, artistCounts, songCounts)
	if err != nil {
		return nil, false, err
	}
	challengeRanked, err := u.calculator.Calculate(condition, candidates, entity.RecommendationTypeDailyChallenge, artistCounts, songCounts)
	if err != nil {
		return nil, false, err
	}

	first, second, ok := u.selectMatches(matchRanked)
	if !ok {
		return nil, false, entity.ErrDailyDiscoveryCandidatesNotFound
	}
	third, challengeStage, ok := u.selectChallenge(
		challengeRanked,
		condition,
		artistCounts,
		map[uuid.UUID]bool{first.Data.Song.ID: true, second.Data.Song.ID: true},
		map[uuid.UUID]bool{first.Data.Artist.ID: true, second.Data.Artist.ID: true},
	)
	if !ok {
		return nil, false, entity.ErrDailyDiscoveryCandidatesNotFound
	}

	firstDetail, err := dailyDetail(first, entity.RecommendationTypeDailyMatchOne, 1, dailyReason(entity.RecommendationTypeDailyMatchOne, first), false)
	if err != nil {
		return nil, false, err
	}
	secondDetail, err := dailyDetail(second, entity.RecommendationTypeDailyMatchTwo, 2, dailyReason(entity.RecommendationTypeDailyMatchTwo, second), false)
	if err != nil {
		return nil, false, err
	}
	thirdDetail, err := dailyDetail(third, entity.RecommendationTypeDailyChallenge, 3, dailyReason(entity.RecommendationTypeDailyChallenge, third), challengeStage > 0)
	if err != nil {
		return nil, false, err
	}
	value := &entity.Recommendation{
		UserID:              userID,
		Source:              entity.RecommendationSourceDailyDiscovery,
		TargetDate:          &targetDate,
		RecommendedAt:       now,
		AlgorithmVersion:    u.config.AlgorithmVersion,
		WeightConfigVersion: u.config.WeightConfigVersion,
		Details:             []entity.RecommendationDetail{firstDetail, secondDetail, thirdDetail},
	}

	// 保存直前にも同日データを確認し、Application側でも重複生成を抑止する。
	if existing, findErr := u.recommendations.FindDailyByDate(ctx, userID, targetDate); findErr == nil {
		return existing, false, nil
	} else if !errors.Is(findErr, entity.ErrDailyDiscoveryNotFound) {
		return nil, false, entity.ErrDailyDiscoveryFetchFailed
	}

	err = u.tx.WithinTransaction(ctx, func(tx repository.ITransaction) error {
		recentSongIDs, findErr := u.recommendations.FindRecommendedSongIDsWithTransaction(
			ctx,
			tx,
			userID,
			now.AddDate(0, 0, -u.config.DuplicateDays),
			now,
		)
		if findErr != nil {
			return findErr
		}
		if containsRecommendationSong(value.Details, recentSongIDs) {
			return entity.ErrDuplicateRecommendationSong
		}
		return u.recommendations.Create(ctx, tx, value)
	})
	if err != nil {
		if saved, fetchErr := u.recommendations.FindDailyByDate(ctx, userID, targetDate); fetchErr == nil {
			return saved, false, nil
		}
		if errors.Is(err, entity.ErrConflict) {
			return nil, false, entity.ErrDailyDiscoveryGenerationConflict
		}
		if errors.Is(err, entity.ErrDuplicateRecommendationSong) {
			return nil, false, err
		}
		return nil, false, entity.ErrDailyDiscoverySaveFailed
	}

	saved, err := u.recommendations.FindDailyByDate(ctx, userID, targetDate)
	if err != nil {
		return nil, false, entity.ErrDailyDiscoveryFetchFailed
	}
	return saved, true, nil
}

func (u *dailyDiscoveryUsecase) selectMatches(ranked []SongCandidate) (SongCandidate, SongCandidate, bool) {
	eligible := make([]SongCandidate, 0, len(ranked))
	for _, candidate := range ranked {
		matchScore := candidate.Score.Mood + candidate.Score.Element + candidate.Score.Atmosphere + candidate.Score.Genre
		if matchScore >= u.config.MinimumMatchScore {
			eligible = append(eligible, candidate)
		}
	}
	if len(eligible) < 2 {
		return SongCandidate{}, SongCandidate{}, false
	}
	first := eligible[0]
	for _, candidate := range eligible[1:] {
		if candidate.Data.Artist.ID != first.Data.Artist.ID {
			return first, candidate, true
		}
	}
	return first, eligible[1], true
}

func (u *dailyDiscoveryUsecase) selectChallenge(
	ranked []SongCandidate,
	condition RecommendationCondition,
	artistCounts map[uuid.UUID]int64,
	selectedSongs map[uuid.UUID]bool,
	selectedArtists map[uuid.UUID]bool,
) (SongCandidate, int, bool) {
	// Discovery条件は strict -> relaxed の順で緩和する。ただし3曲のSong重複と
	// Mood/Element/Atmosphereの最低接点は最後まで緩和しない。
	for stage := 0; stage < 3; stage++ {
		for _, candidate := range ranked {
			if selectedSongs[candidate.Data.Song.ID] {
				continue
			}
			if !hasMinimumContact(candidate, condition) {
				continue
			}
			seenArtist := artistCounts[candidate.Data.Artist.ID] > 0
			if !matchesDiscoveryStage(candidate.Data, condition, seenArtist, stage) {
				continue
			}
			if stage == 0 && selectedArtists[candidate.Data.Artist.ID] {
				continue
			}
			return candidate, stage, true
		}
	}
	return SongCandidate{}, 0, false
}

func dailyDetail(candidate SongCandidate, kind entity.RecommendationType, position int, reason string, fallbackUsed bool) (entity.RecommendationDetail, error) {
	breakdown, err := json.Marshal(candidate.Score)
	if err != nil {
		return entity.RecommendationDetail{}, entity.ErrScoreCalculationFailed
	}
	return entity.RecommendationDetail{
		SongID:         candidate.Data.Song.ID,
		Type:           kind,
		Score:          candidate.Score.Total,
		ScoreBreakdown: breakdown,
		Position:       position,
		Reason:         reason,
		FallbackUsed:   fallbackUsed,
	}, nil
}

func dailyReason(kind entity.RecommendationType, candidate SongCandidate) string {
	type reasonPart struct {
		name  string
		score float64
	}
	parts := []reasonPart{
		{name: "気分", score: candidate.Score.Mood},
		{name: "重視する要素", score: candidate.Score.Element},
		{name: "雰囲気", score: candidate.Score.Atmosphere},
		{name: "ジャンル", score: candidate.Score.Genre},
		{name: "アーティスト傾向", score: candidate.Score.Artist},
	}
	sort.SliceStable(parts, func(i, j int) bool { return parts[i].score > parts[j].score })
	matched := make([]string, 0, 2)
	for _, part := range parts {
		if part.score <= 0 {
			continue
		}
		matched = append(matched, part.name)
		if len(matched) == 2 {
			break
		}
	}
	if len(matched) == 0 {
		switch kind {
		case entity.RecommendationTypeDailyChallenge:
			return "好みとの接点を残しながら、新しい要素を楽しめる1曲です"
		default:
			return "あなたの好みに近い今日の1曲です"
		}
	}
	if kind == entity.RecommendationTypeDailyChallenge {
		return fmt.Sprintf("%sとの接点を残しながら、新しい音楽体験を広げる1曲です", joinJapanese(matched))
	}
	return fmt.Sprintf("%sとの一致度が高い今日の1曲です", joinJapanese(matched))
}

func joinJapanese(values []string) string {
	if len(values) == 1 {
		return values[0]
	}
	return values[0] + "と" + values[1]
}

func (u *dailyDiscoveryUsecase) List(ctx context.Context, userID uuid.UUID, page, limit int) (*DailyDiscoveryList, error) {
	if page < 1 || limit < 1 || limit > 100 {
		return nil, entity.ErrInvalidInput
	}
	items, total, err := u.recommendations.ListDaily(ctx, userID, page, limit)
	if err != nil {
		return nil, entity.ErrDailyDiscoveryFetchFailed
	}
	pages := int((total + int64(limit) - 1) / int64(limit))
	return &DailyDiscoveryList{Items: items, Page: page, Limit: limit, Total: total, TotalPages: pages}, nil
}

func (u *dailyDiscoveryUsecase) GetByID(ctx context.Context, userID, id uuid.UUID) (*entity.Recommendation, error) {
	value, err := u.recommendations.FindByID(ctx, id)
	if errors.Is(err, entity.ErrRecommendationNotFound) {
		return nil, entity.ErrDailyDiscoveryNotFound
	}
	if err != nil {
		return nil, entity.ErrDailyDiscoveryFetchFailed
	}
	if value.Source != entity.RecommendationSourceDailyDiscovery {
		return nil, entity.ErrDailyDiscoveryNotFound
	}
	if value.UserID != userID {
		return nil, entity.ErrDailyDiscoveryAccessDenied
	}
	return value, nil
}
