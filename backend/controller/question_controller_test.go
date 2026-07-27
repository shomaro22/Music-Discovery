package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"music-discovery/backend/entity"
	"music-discovery/backend/middleware"
	"music-discovery/backend/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type questionUsecaseControllerStub struct{ err error }

func (s questionUsecaseControllerStub) ListQuestions(context.Context) ([]entity.Question, error) {
	return nil, s.err
}
func (s questionUsecaseControllerStub) SaveAnswers(context.Context, uuid.UUID, usecase.SaveQuestionAnswersInput) (*usecase.SaveQuestionAnswersResult, error) {
	return nil, s.err
}
func (s questionUsecaseControllerStub) GetLatestAnswers(context.Context, uuid.UUID) (*usecase.LatestQuestionAnswersResult, error) {
	return nil, s.err
}

func TestQuestionControllerSaveAnswersErrorCodes(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"required", entity.ErrRequiredQuestionMissing, 400, "REQUIRED_QUESTION_MISSING"},
		{"duplicate question", entity.ErrDuplicateQuestion, 400, "DUPLICATE_QUESTION"},
		{"selection", entity.ErrInvalidSelectionCount, 400, "INVALID_SELECTION_COUNT"},
		{"duplicate choice", entity.ErrDuplicateChoice, 400, "DUPLICATE_CHOICE"},
		{"question missing", entity.ErrQuestionNotFound, 404, "QUESTION_NOT_FOUND"},
		{"choice missing", entity.ErrChoiceNotFound, 404, "CHOICE_NOT_FOUND"},
		{"mismatch", entity.ErrQuestionChoiceMismatch, 409, "QUESTION_CHOICE_MISMATCH"},
		{"suspended", entity.ErrUserSuspended, 403, "USER_SUSPENDED"},
		{"answer save", entity.ErrAnswerSaveFailed, 500, "ANSWER_SAVE_FAILED"},
		{"profile", entity.ErrProfileCreateFailed, 500, "PROFILE_CREATE_FAILED"},
		{"internal", errors.New("boom"), 500, "INTERNAL_SERVER_ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/api/user-answers", strings.NewReader(`{"answers":[]}`))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			ctx := e.NewContext(req, rec)
			ctx.Set(middleware.AuthUserIDKey, uuid.New())
			controller := NewQuestionController(questionUsecaseControllerStub{err: tt.err})
			if err := controller.SaveAnswers(ctx); err != nil {
				t.Fatalf("SaveAnswers returned error: %v", err)
			}
			if rec.Code != tt.status {
				t.Fatalf("status=%d want=%d body=%s", rec.Code, tt.status, rec.Body.String())
			}
			var body map[string]interface{}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body["code"] != tt.code {
				t.Fatalf("code=%v want=%s", body["code"], tt.code)
			}
		})
	}
}

func TestQuestionControllerRequiresAuthentication(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/user-answers", strings.NewReader(`{"answers":[]}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	controller := NewQuestionController(questionUsecaseControllerStub{})
	if err := controller.SaveAnswers(ctx); err != nil {
		t.Fatalf("SaveAnswers returned error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "UNAUTHORIZED") {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
