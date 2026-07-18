package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type fakeUserUsecase struct {
	user  *entity.User
	err   error
	input usecase.SignUpInput
}

func (f *fakeUserUsecase) SignUp(_ context.Context, input usecase.SignUpInput) (*entity.User, error) {
	f.input = input
	return f.user, f.err
}

func runSignupRequest(t *testing.T, body string, uc usecase.IUserUsecase) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderXRequestID, "req-123")
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Response().Header().Set(echo.HeaderXRequestID, "req-123")
	_ = NewUserController(uc).SignUp(ctx)
	return rec
}

func TestSignupControllerSuccess(t *testing.T) {
	user := &entity.User{ID: uuid.New(), Name: "Taro", Email: "taro@example.com", Role: entity.UserRoleUser, Status: entity.UserStatusActive, CreatedAt: time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)}
	uc := &fakeUserUsecase{user: user}
	rec := runSignupRequest(t, `{"name":" Taro ","email":" TARO@example.com ","password":"password"}`, uc)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if uc.input.Name != "Taro" || uc.input.Email != "taro@example.com" {
		t.Fatalf("input=%#v", uc.input)
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("password")) {
		t.Fatalf("password leaked: %s", rec.Body.String())
	}
}

func TestSignupControllerErrors(t *testing.T) {
	tests := []struct {
		name, body string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"bad-json", `{`, nil, 400, "INVALID_INPUT"},
		{"validation", `{"name":"","email":"bad","password":"short"}`, nil, 400, "INVALID_INPUT"},
		{"duplicate", `{"name":"Taro","email":"taro@example.com","password":"password"}`, entity.ErrEmailAlreadyExists, 409, "EMAIL_ALREADY_EXISTS"},
		{"internal", `{"name":"Taro","email":"taro@example.com","password":"password"}`, errors.New("boom"), 500, "INTERNAL_ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &fakeUserUsecase{err: tt.err}
			rec := runSignupRequest(t, tt.body, uc)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			var apiErr entity.APIErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &apiErr); err != nil {
				t.Fatal(err)
			}
			if apiErr.Code != tt.wantCode || apiErr.RequestID != "req-123" {
				t.Fatalf("error=%#v", apiErr)
			}
			var raw map[string]any
			_ = json.Unmarshal(rec.Body.Bytes(), &raw)
			if len(raw) != 4 {
				t.Fatalf("error response fields=%d body=%s", len(raw), rec.Body.String())
			}
		})
	}
}
