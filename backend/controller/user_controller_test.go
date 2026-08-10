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
	user           *entity.User
	err            error
	input          usecase.SignUpInput
	refreshResult  *usecase.RefreshResult
	refreshInput   string
	refreshCalls   int
	logoutInput    string
	logoutCalls    int
	withdrawInput  usecase.WithdrawInput
	withdrawUserID uuid.UUID
	withdrawCalls  int
}

func (f *fakeUserUsecase) SignUp(_ context.Context, input usecase.SignUpInput) (*usecase.SignUpResult, error) {
	f.input = input
	if f.err != nil {
		return nil, f.err
	}
	if f.user == nil {
		return nil, entity.ErrInvalidInput
	}
	return &usecase.SignUpResult{
		AccessToken:           "access-token",
		ExpiresIn:             300,
		RefreshToken:          "refresh-token",
		RefreshTokenExpiresAt: time.Now().Add(time.Hour),
		CSRFToken:             "csrf-token",
		User:                  f.user,
	}, nil
}

func (f *fakeUserUsecase) Login(context.Context, usecase.LoginInput) (*usecase.LoginResult, error) {
	return nil, f.err
}

func (f *fakeUserUsecase) Refresh(_ context.Context, token string) (*usecase.RefreshResult, error) {
	f.refreshCalls++
	f.refreshInput = token
	return f.refreshResult, f.err
}
func (f *fakeUserUsecase) Logout(_ context.Context, token string) error {
	f.logoutCalls++
	f.logoutInput = token
	return f.err
}
func (f *fakeUserUsecase) CurrentUser(context.Context, uuid.UUID) (*usecase.CurrentUserResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.user == nil {
		return nil, entity.ErrInvalidCredentials
	}
	return &usecase.CurrentUserResult{ID: f.user.ID, Name: f.user.Name, Email: f.user.Email, Role: f.user.Role, Status: f.user.Status}, nil
}
func (f *fakeUserUsecase) ValidateTokenVersion(context.Context, uuid.UUID, uint64) (*entity.User, error) {
	return f.user, f.err
}
func (f *fakeUserUsecase) Withdraw(_ context.Context, userID uuid.UUID, input usecase.WithdrawInput) (*usecase.WithdrawResult, error) {
	f.withdrawCalls++
	f.withdrawUserID = userID
	f.withdrawInput = input
	if f.err != nil {
		return nil, f.err
	}
	return &usecase.WithdrawResult{Message: "退会が完了しました。"}, nil
}

func (f *fakeUserUsecase) UpdateProfile(_ context.Context, userID uuid.UUID, input usecase.UpdateProfileInput) (*usecase.CurrentUserResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.user == nil {
		return nil, entity.ErrInvalidCredentials
	}
	return &usecase.CurrentUserResult{
		ID:     userID,
		Name:   input.Name,
		Email:  input.Email,
		Role:   f.user.Role,
		Status: f.user.Status,
	}, nil
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

	var apiErr apiErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &apiErr); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if apiErr.Code != wantCode {
		t.Fatalf("code=%q want=%q body=%s", apiErr.Code, wantCode, rec.Body.String())
	}
	if apiErr.RequestID != "req-123" {
		t.Fatalf("request_id=%q want=%q", apiErr.RequestID, "req-123")
	}

	decoder := json.NewDecoder(bytes.NewReader(rec.Body.Bytes()))
	decoder.DisallowUnknownFields()
	var exactResponse apiErrorResponse
	if err := decoder.Decode(&exactResponse); err != nil {
		t.Fatalf("unexpected response field: %v body=%s", err, rec.Body.String())
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
	if csrfCookie == nil || csrfCookie.Value != "csrf-new" || csrfCookie.Path != "/" || !csrfCookie.HttpOnly {
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

func runLogoutRequest(t *testing.T, uc usecase.IUserUsecase, withCookie bool) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	if withCookie {
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "refresh-token"})
	}
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	_ = NewUserController(uc).Logout(ctx)
	return rec
}

func TestLogoutControllerSuccessClearsCookies(t *testing.T) {
	uc := &fakeUserUsecase{}
	rec := runLogoutRequest(t, uc, true)
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if uc.logoutInput != "refresh-token" || uc.logoutCalls != 1 {
		t.Fatalf("input=%q calls=%d", uc.logoutInput, uc.logoutCalls)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("cookies=%d", len(cookies))
	}
	for _, cookie := range cookies {
		if cookie.Value != "" || cookie.MaxAge >= 0 {
			t.Fatalf("cookie=%+v", cookie)
		}
	}
}

func TestLogoutControllerWithoutCookieIsSuccessful(t *testing.T) {
	uc := &fakeUserUsecase{}
	rec := runLogoutRequest(t, uc, false)
	if rec.Code != http.StatusNoContent || uc.logoutInput != "" {
		t.Fatalf("status=%d input=%q", rec.Code, uc.logoutInput)
	}
}

func TestLogoutControllerInternalErrorDoesNotClearCookies(t *testing.T) {
	rec := runLogoutRequest(t, &fakeUserUsecase{err: errors.New("db down")}, true)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Fatalf("cookies should not be cleared on backend failure")
	}
}

func TestCurrentUserControllerSuccess(t *testing.T) {
	userID := uuid.New()
	uc := &fakeUserUsecase{user: &entity.User{
		ID: userID, Name: "Taro", Email: "taro@example.com",
		Role: entity.UserRoleUser, Status: entity.UserStatusActive,
		PasswordHash: "must-not-leak", TokenVersion: 4,
	}}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Set("auth_user_id", userID)

	if err := NewUserController(uc).CurrentUser(ctx); err != nil {
		t.Fatalf("err=%v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("must-not-leak")) || bytes.Contains(rec.Body.Bytes(), []byte("token_version")) {
		t.Fatalf("private field leaked: %s", rec.Body.String())
	}
}

func TestCurrentUserControllerRequiresContextUserID(t *testing.T) {
	uc := &fakeUserUsecase{}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	_ = NewUserController(uc).CurrentUser(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func runWithdrawRequest(t *testing.T, body string, uc usecase.IUserUsecase, withUserID bool) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/users/me", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderXRequestID, "req-123")
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Response().Header().Set(echo.HeaderXRequestID, "req-123")
	if withUserID {
		ctx.Set("auth_user_id", uuid.New())
	}
	_ = NewUserController(uc).Withdraw(ctx)
	return rec
}

func TestWithdrawControllerSuccessClearsCookiesAndReturnsOnlyMessage(t *testing.T) {
	uc := &fakeUserUsecase{}
	rec := runWithdrawRequest(t, `{"current_password":"password"}`, uc, true)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if uc.withdrawCalls != 1 || uc.withdrawInput.CurrentPassword != "password" || uc.withdrawUserID == uuid.Nil {
		t.Fatalf("unexpected usecase call: calls=%d userID=%v input=%#v", uc.withdrawCalls, uc.withdrawUserID, uc.withdrawInput)
	}
	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response) != 1 || response["message"] == "" {
		t.Fatalf("unexpected response: %v", response)
	}
	for _, value := range []string{"password", "password_hash", "access_token", "refresh_token", "token_version"} {
		if bytes.Contains(rec.Body.Bytes(), []byte(value)) {
			t.Fatalf("sensitive value %q leaked: %s", value, rec.Body.String())
		}
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("cookies=%d want=2", len(cookies))
	}
	for _, cookie := range cookies {
		if cookie.Value != "" || cookie.MaxAge >= 0 {
			t.Fatalf("cookie not cleared: %+v", cookie)
		}
	}
}

func TestWithdrawControllerRejectsInvalidRequests(t *testing.T) {
	for _, tc := range []struct {
		name       string
		body       string
		withUserID bool
		status     int
	}{
		{name: "missing auth user", body: `{"current_password":"password"}`, status: http.StatusUnauthorized},
		{name: "invalid json", body: `{`, withUserID: true, status: http.StatusBadRequest},
		{name: "empty password", body: `{"current_password":""}`, withUserID: true, status: http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			uc := &fakeUserUsecase{}
			rec := runWithdrawRequest(t, tc.body, uc, tc.withUserID)
			if rec.Code != tc.status {
				t.Fatalf("status=%d want=%d body=%s", rec.Code, tc.status, rec.Body.String())
			}
			if uc.withdrawCalls != 0 {
				t.Fatalf("withdraw calls=%d want=0", uc.withdrawCalls)
			}
			if len(rec.Result().Cookies()) != 0 {
				t.Fatal("cookies must not be cleared")
			}
		})
	}
}

func TestWithdrawControllerMapsErrorsWithoutClearingCookies(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "password mismatch", err: entity.ErrCurrentPasswordMismatch, status: http.StatusUnauthorized, code: "CURRENT_PASSWORD_MISMATCH"},
		{name: "already withdrawn", err: entity.ErrInvalidCredentials, status: http.StatusUnauthorized, code: "UNAUTHORIZED"},
		{name: "invalid input", err: entity.ErrInvalidInput, status: http.StatusBadRequest, code: "INVALID_INPUT"},
		{name: "internal", err: entity.ErrInternal, status: http.StatusInternalServerError, code: "INTERNAL_ERROR"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := runWithdrawRequest(t, `{"current_password":"password"}`, &fakeUserUsecase{err: tc.err}, true)
			assertAPIError(t, rec, tc.status, tc.code)
			if len(rec.Result().Cookies()) != 0 {
				t.Fatal("cookies must not be cleared on failure")
			}
		})
	}
}

func TestSignupControllerProductionCookiesAreSecureAndStrict(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("COOKIE_SAME_SITE", "strict")
	user := &entity.User{
		ID:     uuid.New(),
		Name:   "Cookie User",
		Email:  "cookie@example.com",
		Role:   entity.UserRoleUser,
		Status: entity.UserStatusActive,
	}
	rec := runSignupRequest(t, `{"name":"Cookie User","email":"cookie@example.com","password":"Password123!"}`, &fakeUserUsecase{user: user})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("cookies=%d want=2", len(cookies))
	}
	for _, cookie := range cookies {
		if !cookie.Secure {
			t.Fatalf("cookie %s must be Secure in production", cookie.Name)
		}
		if cookie.SameSite != http.SameSiteStrictMode {
			t.Fatalf("cookie %s SameSite=%v want Strict", cookie.Name, cookie.SameSite)
		}
	}
	for _, cookie := range cookies {
		if cookie.Name == "refresh_token" && !cookie.HttpOnly {
			t.Fatal("refresh_token must be HttpOnly")
		}
	}
}

func TestSignupControllerSameSiteNoneForCrossSiteProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("COOKIE_SAME_SITE", "none")
	user := &entity.User{ID: uuid.New(), Name: "Cross Site", Email: "cross@example.com", Role: entity.UserRoleUser, Status: entity.UserStatusActive}
	rec := runSignupRequest(t, `{"name":"Cross Site","email":"cross@example.com","password":"Password123!"}`, &fakeUserUsecase{user: user})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	for _, cookie := range rec.Result().Cookies() {
		if cookie.SameSite != http.SameSiteNoneMode || !cookie.Secure || !cookie.Partitioned {
			t.Fatalf("cookie %s SameSite=%v Secure=%v Partitioned=%v", cookie.Name, cookie.SameSite, cookie.Secure, cookie.Partitioned)
		}
		if cookie.Name == "csrf_token" && (!cookie.HttpOnly || cookie.Path != "/") {
			t.Fatalf("csrf cookie=%+v", cookie)
		}
	}
}
