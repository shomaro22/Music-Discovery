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
)

type recommendationUsecaseStub struct {
	value *entity.Recommendation
	err   error
}

func (s *recommendationUsecaseStub) Generate(context.Context, uuid.UUID) (*entity.Recommendation, error) {
	return s.value, s.err
}
func (s *recommendationUsecaseStub) GetLatest(context.Context, uuid.UUID) (*entity.Recommendation, error) {
	return s.value, s.err
}
func (s *recommendationUsecaseStub) GetByID(context.Context, uuid.UUID, uuid.UUID) (*entity.Recommendation, error) {
	return s.value, s.err
}

func TestRecommendationControllerGenerateReturnsCreated(t *testing.T) {
	value := &entity.Recommendation{ID: uuid.New(), RecommendedAt: time.Now().UTC(), Details: []entity.RecommendationDetail{}}
	controller := NewRecommendationController(&recommendationUsecaseStub{value: value})
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/recommendations", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Set(middleware.AuthUserIDKey, uuid.New())
	if err := controller.Generate(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d want=%d", rec.Code, http.StatusCreated)
	}
}

func TestRecommendationControllerMapsErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"answers required", entity.ErrQuestionAnswerRequired, 400, "QUESTION_ANSWER_REQUIRED"},
		{"profile required", entity.ErrUserProfileRequired, 400, "USER_PROFILE_REQUIRED"},
		{"suspended", entity.ErrUserSuspended, 403, "USER_SUSPENDED"},
		{"access denied", entity.ErrRecommendationAccessDenied, 403, "RECOMMENDATION_ACCESS_DENIED"},
		{"not found", entity.ErrRecommendationNotFound, 404, "RECOMMENDATION_NOT_FOUND"},
		{"already generated", entity.ErrRecommendationAlreadyGenerated, 409, "RECOMMENDATION_ALREADY_GENERATED"},
		{"candidates missing", entity.ErrRecommendationCandidatesNotFound, 422, "RECOMMENDATION_CANDIDATES_NOT_FOUND"},
		{"save failed", entity.ErrRecommendationSaveFailed, 500, "RECOMMENDATION_SAVE_FAILED"},
		{"generate failed", entity.ErrRecommendationGenerateFailed, 500, "RECOMMENDATION_GENERATE_FAILED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := NewRecommendationController(&recommendationUsecaseStub{err: tt.err})
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/api/recommendations", nil)
			rec := httptest.NewRecorder()
			ctx := e.NewContext(req, rec)
			ctx.Set(middleware.AuthUserIDKey, uuid.New())
			if err := controller.Generate(ctx); err != nil {
				t.Fatal(err)
			}
			if rec.Code != tt.status {
				t.Fatalf("status=%d want=%d", rec.Code, tt.status)
			}
			var body struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Code != tt.code {
				t.Fatalf("code=%q want=%q", body.Code, tt.code)
			}
		})
	}
}

func TestRecommendationControllerRejectsInvalidID(t *testing.T) {
	controller := NewRecommendationController(&recommendationUsecaseStub{})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/recommendations/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.SetPath("/api/recommendations/:recommendationId")
	ctx.SetParamNames("recommendationId")
	ctx.SetParamValues("not-a-uuid")
	ctx.Set(middleware.AuthUserIDKey, uuid.New())
	if err := controller.GetByID(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d", rec.Code, http.StatusBadRequest)
	}
}
