package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"music-discovery/backend/entity"
	"music-discovery/backend/middleware"
	"music-discovery/backend/usecase"
)

type recommendationHistoryUsecaseStub struct {
	listResult *usecase.RecommendationHistoryResult
	listErr    error
	value      *entity.Recommendation
	timezone   string
	getErr     error
}

func (s *recommendationHistoryUsecaseStub) List(context.Context, uuid.UUID, usecase.RecommendationHistoryInput) (*usecase.RecommendationHistoryResult, error) {
	return s.listResult, s.listErr
}
func (s *recommendationHistoryUsecaseStub) GetByID(context.Context, uuid.UUID, uuid.UUID) (*entity.Recommendation, string, error) {
	return s.value, s.timezone, s.getErr
}

func TestRecommendationHistoryControllerListIncludesTypeReasonAndTimezone(t *testing.T) {
	status := entity.FeedbackStatusLike
	result := &usecase.RecommendationHistoryResult{Page: 1, Limit: 10, Total: 1, TotalPages: 1, Timezone: "Asia/Tokyo", Items: []entity.Recommendation{{
		ID: uuid.New(), Source: entity.RecommendationSourceNormal, RecommendedAt: time.Now(), Details: []entity.RecommendationDetail{{
			ID: uuid.New(), Type: entity.RecommendationTypeYourMatch, Position: 1, Reason: "好みに一致", FeedbackStatus: &status, Song: entity.Song{ID: uuid.New(), Name: "Song", ArtistID: uuid.New(), Artist: entity.Artist{Name: "Artist"}},
		}},
	}}}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/recommendation-history", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Set(middleware.AuthUserIDKey, uuid.New())
	if err := NewRecommendationHistoryController(&recommendationHistoryUsecaseStub{listResult: result}).List(ctx); err != nil {
		t.Fatal(err)
	}
	var body RecommendationHistoryListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Timezone != "Asia/Tokyo" || body.Items[0].Songs[0].RecommendationType != entity.RecommendationTypeYourMatch || body.Items[0].Songs[0].Reason != "好みに一致" {
		t.Fatalf("body=%+v", body)
	}
}

func TestRecommendationHistoryControllerRejectsInvalidPage(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/recommendation-history?page=x", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Set(middleware.AuthUserIDKey, uuid.New())
	if err := NewRecommendationHistoryController(&recommendationHistoryUsecaseStub{}).List(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
