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
	user          *entity.User
	err           error
	input         usecase.SignUpInput
	refreshResult *usecase.RefreshResult
	refreshInput  string
	refreshCalls  int
}

func (f *fakeUserUsecase) SignUp(_ context.Context, input usecase.SignUpInput) (*entity.User, error) {
	f.input = input
	if f.err != nil {
		return nil, f.err
	}
	if f.user == nil {
		return nil, entity.ErrInvalidInput
	}
	return f.user, nil
}

func (f *fakeUserUsecase) Login(context.Context, usecase.LoginInput) (*usecase.LoginResult, error) {
	return nil, f.err
}

func (f *fakeUserUsecase) Refresh(_ context.Context, token string) (*usecase.RefreshResult, error) {
	f.refreshCalls++
	f.refreshInput = token
	return f.refreshResult, f.err
}
func (f *fakeUserUsecase) ValidateTokenVersion(context.Context, uuid.UUID, uint64) (*entity.User, error) {
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
	if uc.input.Name != " Taro " || uc.input.Email != " TARO@example.com " {
		t.Fatalf("controller must pass bound values unchanged: %#v", uc.input)
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

func runRefreshRequest(t *testing.T, uc usecase.IUserUsecase, withCookie bool) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	if withCookie {
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "refresh-old"})
	}
	req.Header.Set(echo.HeaderXRequestID, "req-123")
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Response().Header().Set(echo.HeaderXRequestID, "req-123")
	_ = NewUserController(uc).Refresh(ctx)
	return rec
}

func TestRefreshControllerSuccessUpdatesCookies(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	expiresAt := time.Now().UTC().Add(time.Hour)
	uc := &fakeUserUsecase{refreshResult: &usecase.RefreshResult{AccessToken: "access-new", ExpiresIn: 300, RefreshToken: "refresh-new", RefreshTokenExpiresAt: expiresAt, CSRFToken: "csrf-new"}}
	rec := runRefreshRequest(t, uc, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if uc.refreshInput != "refresh-old" {
		t.Fatalf("refresh input=%q", uc.refreshInput)
	}
	var response RefreshResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.AccessToken != "access-new" || response.TokenType != "Bearer" || response.ExpiresIn != 300 {
		t.Fatalf("response=%+v", response)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("cookies=%d", len(cookies))
	}
	byName := map[string]*http.Cookie{}
	for _, cookie := range cookies {
		byName[cookie.Name] = cookie
	}
	refreshCookie := byName["refresh_token"]
	csrfCookie := byName["csrf_token"]
	if refreshCookie == nil || refreshCookie.Value != "refresh-new" || refreshCookie.Path != "/api/auth" || !refreshCookie.HttpOnly {
		t.Fatalf("refresh cookie=%+v", refreshCookie)
	}
	if csrfCookie == nil || csrfCookie.Value != "csrf-new" || csrfCookie.Path != "/" || csrfCookie.HttpOnly {
		t.Fatalf("csrf cookie=%+v", csrfCookie)
	}
}

func TestRefreshControllerMissingCookieReturns401(t *testing.T) {
	rec := runRefreshRequest(t, &fakeUserUsecase{}, false)
	assertAPIError(t, rec, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN")
}

func TestRefreshControllerErrorsAndCookieDeletion(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		status  int
		code    string
		deletes bool
	}{
		{name: "invalid", err: entity.ErrInvalidRefreshToken, status: http.StatusUnauthorized, code: "INVALID_REFRESH_TOKEN", deletes: true},
		{name: "reused", err: entity.ErrRefreshTokenReused, status: http.StatusUnauthorized, code: "REFRESH_TOKEN_REUSED", deletes: true},
		{name: "suspended", err: entity.ErrUserSuspended, status: http.StatusForbidden, code: "USER_SUSPENDED", deletes: true},
		{name: "internal", err: errors.New("boom"), status: http.StatusInternalServerError, code: "INTERNAL_ERROR", deletes: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := runRefreshRequest(t, &fakeUserUsecase{err: tt.err}, true)
			assertAPIError(t, rec, tt.status, tt.code)
			cookies := rec.Result().Cookies()
			if tt.deletes {
				if len(cookies) != 2 {
					t.Fatalf("cookies=%d", len(cookies))
				}
				for _, cookie := range cookies {
					if cookie.Value != "" || cookie.MaxAge >= 0 || !cookie.Expires.Before(time.Now()) {
						t.Fatalf("cookie not deleted: %+v", cookie)
					}
					if cookie.Name == "refresh_token" && cookie.Path != "/api/auth" {
						t.Fatalf("refresh path=%q", cookie.Path)
					}
					if cookie.Name == "csrf_token" && cookie.Path != "/" {
						t.Fatalf("csrf path=%q", cookie.Path)
					}
				}
			} else if len(cookies) != 0 {
				t.Fatalf("unexpected cookies=%d", len(cookies))
			}
		})
	}
}

func TestRefreshControllerEmptyCookieValueReturns401(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: ""})
	req.Header.Set(echo.HeaderXRequestID, "req-123")
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Response().Header().Set(echo.HeaderXRequestID, "req-123")
	uc := &fakeUserUsecase{}

	_ = NewUserController(uc).Refresh(ctx)

	assertAPIError(t, rec, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN")
	if uc.refreshCalls != 0 {
		t.Fatalf("refresh usecase calls=%d", uc.refreshCalls)
	}
}
