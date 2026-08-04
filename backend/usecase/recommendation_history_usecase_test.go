package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"music-discovery/backend/entity"
	"music-discovery/backend/repository"
)

type historyUserRepoStub struct {
	repository.IUserRepository
	user *entity.User
	err  error
}

func (s *historyUserRepoStub) FindByID(context.Context, uuid.UUID) (*entity.User, error) {
	return s.user, s.err
}

type historyProfileRepoStub struct {
	repository.IUserProfileTimezoneRepository
	timezone string
	err      error
}

func (s *historyProfileRepoStub) FindTimezoneByUserID(context.Context, uuid.UUID) (string, error) {
	return s.timezone, s.err
}

type historyRepoStub struct {
	repository.IRecommendationHistoryRepository
	items     []entity.Recommendation
	total     int64
	value     *entity.Recommendation
	listErr   error
	findErr   error
	condition repository.RecommendationHistoryCondition
}

func (s *historyRepoStub) ListHistory(_ context.Context, condition repository.RecommendationHistoryCondition) ([]entity.Recommendation, int64, error) {
	s.condition = condition
	return s.items, s.total, s.listErr
}
func (s *historyRepoStub) FindByID(context.Context, uuid.UUID) (*entity.Recommendation, error) {
	return s.value, s.findErr
}

func activeHistoryUser(id uuid.UUID) *entity.User {
	return &entity.User{ID: id, Status: entity.UserStatusActive, Role: entity.RoleUser}
}

func TestRecommendationHistoryListValidationAndTimezone(t *testing.T) {
	userID := uuid.New()
	repo := &historyRepoStub{total: 11}
	u := NewRecommendationHistoryUsecase(
		&historyUserRepoStub{user: activeHistoryUser(userID)},
		&historyProfileRepoStub{timezone: "Asia/Tokyo"},
		repo,
	)

	result, err := u.List(context.Background(), userID, RecommendationHistoryInput{Page: 2, Limit: 10, StartDate: "2026-08-01", EndDate: "2026-08-04"})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if result.TotalPages != 2 || result.Timezone != "Asia/Tokyo" {
		t.Fatalf("result=%+v", result)
	}
	if repo.condition.StartAt == nil || repo.condition.EndBefore == nil {
		t.Fatal("date condition was not created")
	}
	if got := repo.condition.EndBefore.In(time.FixedZone("JST", 9*60*60)).Format("2006-01-02"); got != "2026-08-05" {
		t.Fatalf("end before=%s", got)
	}
}

func TestRecommendationHistoryListRejectsInvalidInput(t *testing.T) {
	userID := uuid.New()
	base := func() IRecommendationHistoryUsecase {
		return NewRecommendationHistoryUsecase(&historyUserRepoStub{user: activeHistoryUser(userID)}, &historyProfileRepoStub{}, &historyRepoStub{})
	}
	tests := []struct {
		name  string
		input RecommendationHistoryInput
		want  error
	}{
		{"page", RecommendationHistoryInput{Page: 0, Limit: 10}, entity.ErrInvalidPage},
		{"limit", RecommendationHistoryInput{Page: 1, Limit: 101}, entity.ErrInvalidLimit},
		{"source", RecommendationHistoryInput{Page: 1, Limit: 10, Source: func() *entity.RecommendationSource { v := entity.RecommendationSource("BAD"); return &v }()}, entity.ErrInvalidRecommendationSource},
		{"range", RecommendationHistoryInput{Page: 1, Limit: 10, StartDate: "2026-08-05", EndDate: "2026-08-01"}, entity.ErrInvalidDateRange},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := base().List(context.Background(), userID, tt.input)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err=%v want=%v", err, tt.want)
			}
		})
	}
}

func TestRecommendationHistoryListReturnsFetchErrorWhenTimezoneReadFails(t *testing.T) {
	userID := uuid.New()
	u := NewRecommendationHistoryUsecase(&historyUserRepoStub{user: activeHistoryUser(userID)}, &historyProfileRepoStub{err: entity.ErrInternal}, &historyRepoStub{})
	_, err := u.List(context.Background(), userID, RecommendationHistoryInput{Page: 1, Limit: 10})
	if !errors.Is(err, entity.ErrRecommendationHistoryFetchFailed) {
		t.Fatalf("err=%v", err)
	}
}

func TestRecommendationHistoryGetByIDChecksOwnerAndDetailCount(t *testing.T) {
	userID := uuid.New()
	otherID := uuid.New()
	value := &entity.Recommendation{ID: uuid.New(), UserID: otherID, Details: make([]entity.RecommendationDetail, 3)}
	u := NewRecommendationHistoryUsecase(&historyUserRepoStub{user: activeHistoryUser(userID)}, &historyProfileRepoStub{}, &historyRepoStub{value: value})
	_, _, err := u.GetByID(context.Background(), userID, value.ID)
	if !errors.Is(err, entity.ErrRecommendationHistoryAccessDenied) {
		t.Fatalf("err=%v", err)
	}

	value.UserID = userID
	value.Details = value.Details[:2]
	_, _, err = u.GetByID(context.Background(), userID, value.ID)
	if !errors.Is(err, entity.ErrRecommendationHistoryFetchFailed) {
		t.Fatalf("err=%v", err)
	}
}
