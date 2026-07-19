package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type fakeUserUsecase struct {
	user        *entity.User
	err         error
	input       usecase.SignUpInput
	loginInput  usecase.LoginInput
	loginResult *usecase.LoginResult
}

func (f *fakeUserUsecase) SignUp(_ context.Context, input usecase.SignUpInput) (*entity.User, error) {
	f.input = input
	return f.user, f.err
}

func (f *fakeUserUsecase) Refresh(context.Context, string) (*usecase.RefreshResult, error) {
	return nil, f.err
}
func (f *fakeUserUsecase) Login(_ context.Context, input usecase.LoginInput) (*usecase.LoginResult, error) {
	f.loginInput = input
	return f.loginResult, f.err
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

func runLoginRequest(t *testing.T, body string, uc usecase.IUserUsecase, cfg CookieConfig) (*httptest.ResponseRecorder, error) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Response().Header().Set(echo.HeaderXRequestID, "req-login")
	err := NewUserController(uc, cfg).Login(ctx)
	if err != nil {
		e.HTTPErrorHandler = func(err error, c echo.Context) { NewTestHTTPErrorHandler()(err, c) }
		e.HTTPErrorHandler(err, ctx)
	}
	return rec, err
}

func NewTestHTTPErrorHandler() echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		status := 500
		var he *echo.HTTPError
		if errors.As(err, &he) {
			status = he.Code
		}
		_ = c.JSON(status, entity.APIErrorResponse{Status: status, Code: "INTERNAL_ERROR", Message: "サーバー内部でエラーが発生しました", RequestID: c.Response().Header().Get(echo.HeaderXRequestID)})
	}
}

func TestLoginControllerSuccessCookiesAndResponse(t *testing.T) {
	expires := time.Now().Add(time.Hour)
	user := &entity.User{ID: uuid.New(), Name: "Taro", Email: "taro@example.com", Role: entity.UserRoleUser, Status: entity.UserStatusActive, PasswordHash: "secret-hash", TokenVersion: 5}
	uc := &fakeUserUsecase{loginResult: &usecase.LoginResult{AccessToken: "access", ExpiresIn: 300, RefreshToken: "refresh-secret", RefreshTokenExpiresAt: expires, CSRFToken: "csrf-secret", User: user}}
	cfg := CookieConfig{RefreshName: "rt", CSRFName: "csrf", Domain: "example.com", Path: "/api/auth", Secure: true, SameSite: http.SameSiteNoneMode}
	rec, _ := runLoginRequest(t, `{"email":" TARO@EXAMPLE.COM ","password":" password "}`, uc, cfg)
	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if uc.loginInput.Email != "taro@example.com" || uc.loginInput.Password != " password " {
		t.Fatalf("input=%#v", uc.loginInput)
	}
	body := rec.Body.String()
	for _, secret := range []string{"refresh-secret", "csrf-secret", "secret-hash", "token_version", "password_hash"} {
		if strings.Contains(body, secret) {
			t.Fatalf("secret leaked: %s", body)
		}
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("cookies=%d", len(cookies))
	}
	var rt, csrf *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "rt" {
			rt = cookie
		}
		if cookie.Name == "csrf" {
			csrf = cookie
		}
	}
	if rt == nil || !rt.HttpOnly || !rt.Secure || rt.SameSite != http.SameSiteNoneMode || rt.Path != "/api/auth" || rt.MaxAge <= 0 {
		t.Fatalf("refresh cookie=%#v", rt)
	}
	if csrf == nil || csrf.HttpOnly || !csrf.Secure || csrf.SameSite != http.SameSiteNoneMode || csrf.MaxAge <= 0 {
		t.Fatalf("csrf cookie=%#v", csrf)
	}
}

func TestLoginControllerErrorsAndNoCookies(t *testing.T) {
	tests := []struct {
		body   string
		err    error
		status int
		code   string
	}{
		{`{`, nil, 400, "INVALID_INPUT"}, {`{"email":"bad","password":"password"}`, nil, 400, "INVALID_INPUT"},
		{`{"email":"a@b.com","password":"password"}`, entity.ErrInvalidCredentials, 401, "INVALID_CREDENTIALS"},
		{`{"email":"a@b.com","password":"password"}`, entity.ErrUserSuspended, 403, "USER_SUSPENDED"},
		{`{"email":"a@b.com","password":"password"}`, errors.New("db"), 500, "INTERNAL_ERROR"},
	}
	cfg := CookieConfig{RefreshName: "rt", CSRFName: "csrf", Path: "/api/auth", SameSite: http.SameSiteLaxMode}
	for _, tt := range tests {
		rec, _ := runLoginRequest(t, tt.body, &fakeUserUsecase{err: tt.err}, cfg)
		if rec.Code != tt.status {
			t.Fatalf("status=%d want=%d body=%s", rec.Code, tt.status, rec.Body.String())
		}
		if len(rec.Result().Cookies()) != 0 {
			t.Fatal("cookies emitted on failure")
		}
		var got entity.APIErrorResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &got)
		if got.Code != tt.code {
			t.Fatalf("code=%s want=%s", got.Code, tt.code)
		}
	}
}
