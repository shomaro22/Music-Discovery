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

func assertAPIError(
	t *testing.T,
	rec *httptest.ResponseRecorder,
	wantStatus int,
	wantCode string,
) {
	t.Helper()

	if rec.Code != wantStatus {
		t.Fatalf("status=%d want=%d body=%s", rec.Code, wantStatus, rec.Body.String())
	}

	var apiErr entity.APIErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &apiErr); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if apiErr.Code != wantCode {
		t.Fatalf("code=%q want=%q body=%s", apiErr.Code, wantCode, rec.Body.String())
	}
	if apiErr.RequestID != "req-123" {
		t.Fatalf("request_id=%q want=%q", apiErr.RequestID, "req-123")
	}

	var raw map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode raw response: %v", err)
	}
	if len(raw) != 4 {
		t.Fatalf("error response fields=%d body=%s", len(raw), rec.Body.String())
	}
}

func TestSignupControllerSuccess(t *testing.T) {
	user := &entity.User{
		ID:        uuid.New(),
		Name:      "Taro",
		Email:     "taro@example.com",
		Role:      entity.UserRoleUser,
		Status:    entity.UserStatusActive,
		CreatedAt: time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC),
	}
	uc := &fakeUserUsecase{user: user}

	rec := runSignupRequest(
		t,
		`{"name":" Taro ","email":" TARO@example.com ","password":"password"}`,
		uc,
	)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if uc.input.Name != "Taro" || uc.input.Email != "taro@example.com" {
		t.Fatalf("input=%#v", uc.input)
	}
	if uc.input.Password != "password" {
		t.Fatalf("password input=%q", uc.input.Password)
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("password")) {
		t.Fatalf("password leaked: %s", rec.Body.String())
	}
}

func TestSignupControllerInvalidJSON(t *testing.T) {
	uc := &fakeUserUsecase{}
	rec := runSignupRequest(t, `{`, uc)

	assertAPIError(t, rec, http.StatusBadRequest, "INVALID_INPUT")
}

func TestSignupControllerNameRequired(t *testing.T) {
	uc := &fakeUserUsecase{}
	rec := runSignupRequest(
		t,
		`{"name":"","email":"taro@example.com","password":"password"}`,
		uc,
	)

	assertAPIError(t, rec, http.StatusBadRequest, "INVALID_INPUT")
}

func TestSignupControllerEmailRequired(t *testing.T) {
	uc := &fakeUserUsecase{}
	rec := runSignupRequest(
		t,
		`{"name":"Taro","email":"","password":"password"}`,
		uc,
	)

	assertAPIError(t, rec, http.StatusBadRequest, "INVALID_INPUT")
}

func TestSignupControllerInvalidEmailFormat(t *testing.T) {
	uc := &fakeUserUsecase{}
	rec := runSignupRequest(
		t,
		`{"name":"Taro","email":"invalid-email","password":"password"}`,
		uc,
	)

	assertAPIError(t, rec, http.StatusBadRequest, "INVALID_INPUT")
}

func TestSignupControllerPasswordRequired(t *testing.T) {
	uc := &fakeUserUsecase{}
	rec := runSignupRequest(
		t,
		`{"name":"Taro","email":"taro@example.com","password":""}`,
		uc,
	)

	assertAPIError(t, rec, http.StatusBadRequest, "INVALID_INPUT")
}

func TestSignupControllerPasswordTooShort(t *testing.T) {
	uc := &fakeUserUsecase{}
	rec := runSignupRequest(
		t,
		`{"name":"Taro","email":"taro@example.com","password":"1234567"}`,
		uc,
	)

	assertAPIError(t, rec, http.StatusBadRequest, "INVALID_INPUT")
}

func TestSignupControllerUsecaseErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "duplicate email",
			err:        entity.ErrEmailAlreadyExists,
			wantStatus: http.StatusConflict,
			wantCode:   "EMAIL_ALREADY_EXISTS",
		},
		{
			name:       "invalid input",
			err:        entity.ErrInvalidInput,
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_INPUT",
		},
		{
			name:       "internal error",
			err:        errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &fakeUserUsecase{err: tt.err}
			rec := runSignupRequest(
				t,
				`{"name":"Taro","email":"taro@example.com","password":"password"}`,
				uc,
			)

			assertAPIError(t, rec, tt.wantStatus, tt.wantCode)
		})
	}
}
