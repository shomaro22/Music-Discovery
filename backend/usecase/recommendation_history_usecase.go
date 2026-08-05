package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"music-discovery/backend/entity"
	"music-discovery/backend/repository"
)

const RecommendationHistoryDefaultLimit = 10
const RecommendationHistoryMaxLimit = 100

type RecommendationHistoryInput struct {
	Page      int
	Limit     int
	Source    *entity.RecommendationSource
	StartDate string
	EndDate   string
}

type RecommendationHistoryResult struct {
	Items      []entity.Recommendation
	Page       int
	Limit      int
	Total      int64
	TotalPages int
	Timezone   string
}

type IRecommendationHistoryUsecase interface {
	List(context.Context, uuid.UUID, RecommendationHistoryInput) (*RecommendationHistoryResult, error)
	GetByID(context.Context, uuid.UUID, uuid.UUID) (*entity.Recommendation, string, error)
}

type recommendationHistoryUsecase struct {
	users           repository.IUserRepository
	profiles        repository.IUserProfileTimezoneRepository
	recommendations repository.IRecommendationHistoryRepository
}

func NewRecommendationHistoryUsecase(users repository.IUserRepository, profiles repository.IUserProfileTimezoneRepository, recommendations repository.IRecommendationHistoryRepository) IRecommendationHistoryUsecase {
	return &recommendationHistoryUsecase{users: users, profiles: profiles, recommendations: recommendations}
}

func (u *recommendationHistoryUsecase) activeUser(ctx context.Context, userID uuid.UUID) error {
	user, err := u.users.FindByID(ctx, userID)
	if err != nil {
		return entity.ErrRecommendationHistoryFetchFailed
	}
	if user.Status == entity.UserStatusSuspended {
		return entity.ErrUserSuspended
	}
	return nil
}

func (u *recommendationHistoryUsecase) location(ctx context.Context, userID uuid.UUID) (*time.Location, error) {
	zone, err := u.profiles.FindTimezoneByUserID(ctx, userID)
	if err != nil && !errors.Is(err, entity.ErrNotFound) {
		return nil, entity.ErrRecommendationHistoryFetchFailed
	}
	if zone == "" {
		zone = "Asia/Tokyo"
	}
	location, err := time.LoadLocation(zone)
	if err != nil {
		return time.FixedZone("Asia/Tokyo", 9*60*60), nil
	}
	return location, nil
}

func historyDate(value string, location *time.Location, end bool) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, location)
	if err != nil {
		return nil, entity.ErrInvalidDateRange
	}
	if end {
		parsed = parsed.AddDate(0, 0, 1)
	}
	return &parsed, nil
}

func (u *recommendationHistoryUsecase) List(ctx context.Context, userID uuid.UUID, input RecommendationHistoryInput) (*RecommendationHistoryResult, error) {
	if err := u.activeUser(ctx, userID); err != nil {
		return nil, err
	}
	if input.Page < 1 {
		return nil, entity.ErrInvalidPage
	}
	if input.Limit < 1 || input.Limit > RecommendationHistoryMaxLimit {
		return nil, entity.ErrInvalidLimit
	}
	if input.Source != nil && *input.Source != entity.RecommendationSourceNormal && *input.Source != entity.RecommendationSourceDailyDiscovery {
		return nil, entity.ErrInvalidRecommendationSource
	}
	location, err := u.location(ctx, userID)
	if err != nil {
		return nil, err
	}
	start, err := historyDate(input.StartDate, location, false)
	if err != nil {
		return nil, err
	}
	end, err := historyDate(input.EndDate, location, true)
	if err != nil {
		return nil, err
	}
	if start != nil && end != nil && !start.Before(*end) {
		return nil, entity.ErrInvalidDateRange
	}
	items, total, err := u.recommendations.ListHistory(ctx, repository.RecommendationHistoryCondition{UserID: userID, Source: input.Source, StartAt: start, EndBefore: end, Page: input.Page, Limit: input.Limit})
	if err != nil {
		return nil, entity.ErrRecommendationHistoryFetchFailed
	}
	pages := 0
	if total > 0 {
		pages = int((total + int64(input.Limit) - 1) / int64(input.Limit))
	}
	return &RecommendationHistoryResult{Items: items, Page: input.Page, Limit: input.Limit, Total: total, TotalPages: pages, Timezone: location.String()}, nil
}

func (u *recommendationHistoryUsecase) GetByID(ctx context.Context, userID, recommendationID uuid.UUID) (*entity.Recommendation, string, error) {
	if recommendationID == uuid.Nil {
		return nil, "", entity.ErrInvalidRecommendationID
	}
	if err := u.activeUser(ctx, userID); err != nil {
		return nil, "", err
	}
	location, err := u.location(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	value, err := u.recommendations.FindByID(ctx, recommendationID)
	if errors.Is(err, entity.ErrRecommendationNotFound) {
		return nil, "", entity.ErrRecommendationHistoryNotFound
	}
	if err != nil {
		return nil, "", entity.ErrRecommendationHistoryFetchFailed
	}
	if value.UserID != userID {
		return nil, "", entity.ErrRecommendationHistoryAccessDenied
	}
	if len(value.Details) != 3 {
		return nil, "", entity.ErrRecommendationHistoryFetchFailed
	}
	return value, location.String(), nil
}
