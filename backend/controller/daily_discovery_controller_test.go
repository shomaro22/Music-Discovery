package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"music-discovery/backend/entity"
	"music-discovery/backend/middleware"
	"music-discovery/backend/usecase"
)

type dailyDiscoveryUsecaseStub struct {
	listPage  int
	listLimit int
	listValue *usecase.DailyDiscoveryList
	value     *entity.Recommendation
	generated bool
	err       error
}

func (s *dailyDiscoveryUsecaseStub) GetToday(context.Context, uuid.UUID) (*entity.Recommendation, bool, error) {
	return s.value, s.generated, s.err
}
func (s *dailyDiscoveryUsecaseStub) List(_ context.Context, _ uuid.UUID, page, limit int) (*usecase.DailyDiscoveryList, error) {
	s.listPage, s.listLimit = page, limit
	if s.listValue == nil {
		s.listValue = &usecase.DailyDiscoveryList{Items: []entity.Recommendation{}, Page: page, Limit: limit}
	}
	return s.listValue, s.err
}
func (s *dailyDiscoveryUsecaseStub) GetByID(context.Context, uuid.UUID, uuid.UUID) (*entity.Recommendation, error) {
	return s.value, s.err
}

func TestDailyDiscoveryControllerListUsesDefaultPaging(t *testing.T) {
	stub := &dailyDiscoveryUsecaseStub{}
	controller := NewDailyDiscoveryController(stub)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/daily-discoveries", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Set(middleware.AuthUserIDKey, uuid.New())

	if err := controller.List(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if stub.listPage != 1 || stub.listLimit != 10 {
		t.Fatalf("page=%d limit=%d", stub.listPage, stub.listLimit)
	}
}

func TestDailyDiscoveryControllerListRejectsInvalidPaging(t *testing.T) {
	tests := []struct {
		query string
		code  string
	}{
		{"?page=0", "INVALID_PAGE"},
		{"?limit=101", "INVALID_LIMIT"},
	}
	for _, test := range tests {
		t.Run(test.code, func(t *testing.T) {
			controller := NewDailyDiscoveryController(&dailyDiscoveryUsecaseStub{})
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/daily-discoveries"+test.query, nil)
			rec := httptest.NewRecorder()
			ctx := e.NewContext(req, rec)
			ctx.Set(middleware.AuthUserIDKey, uuid.New())
			if err := controller.List(ctx); err != nil {
				t.Fatal(err)
			}
			var body struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusBadRequest || body.Code != test.code {
				t.Fatalf("status=%d code=%q", rec.Code, body.Code)
			}
		})
	}
}

func TestDailyDiscoveryControllerMapsDailyErrors(t *testing.T) {
	tests := []struct {
		err    error
		status int
		code   string
	}{
		{entity.ErrDailyDiscoveryGenerateFailed, 500, "DAILY_DISCOVERY_GENERATE_FAILED"},
		{entity.ErrDailyDiscoveryGenerationConflict, 409, "DAILY_DISCOVERY_GENERATION_CONFLICT"},
		{entity.ErrDuplicateRecommendationSong, 409, "DUPLICATE_RECOMMENDATION_SONG"},
		{entity.ErrScoreCalculationFailed, 500, "SCORE_CALCULATION_FAILED"},
		{entity.ErrUserNotFound, 404, "USER_NOT_FOUND"},
		{entity.ErrDailyDiscoveryFetchFailed, 500, "DAILY_DISCOVERY_FETCH_FAILED"},
	}
	for _, test := range tests {
		t.Run(test.code, func(t *testing.T) {
			controller := NewDailyDiscoveryController(&dailyDiscoveryUsecaseStub{err: test.err})
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/daily-discovery", nil)
			rec := httptest.NewRecorder()
			ctx := e.NewContext(req, rec)
			ctx.Set(middleware.AuthUserIDKey, uuid.New())
			if err := controller.GetToday(ctx); err != nil {
				t.Fatal(err)
			}
			var body struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if rec.Code != test.status || body.Code != test.code {
				t.Fatalf("status=%d code=%q", rec.Code, body.Code)
			}
		})
	}
}
