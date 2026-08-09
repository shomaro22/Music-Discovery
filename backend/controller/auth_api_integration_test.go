package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appMiddleware "music-discovery/backend/middleware"
	"music-discovery/backend/migrate"
	"music-discovery/backend/repository"
	testutil "music-discovery/backend/test"
	"music-discovery/backend/usecase"
	"music-discovery/backend/validator"

	"github.com/labstack/echo/v4"
)

func TestAuthAPIIntegrationSignUpLoginAndDuplicate(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-jwt-secret-at-least-32-characters-long")
	t.Setenv("APP_ENV", "test")

	db := testutil.OpenTestDB(t)
	if err := migrate.Run(db, migrate.Config{AppEnvironment: "test"}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	testutil.ResetDatabase(t, db)
	t.Cleanup(func() { testutil.ResetDatabase(t, db) })

	users := repository.NewUserRepository(db)
	refreshTokens := repository.NewRefreshTokenRepository(db)
	tx := repository.NewAuthTransactionManager(db)
	tokens, err := usecase.NewTokenServiceFromEnv()
	if err != nil {
		t.Fatalf("create token service: %v", err)
	}
	userUsecase := usecase.NewUserUsecase(
		users,
		validator.NewUserValidator(),
		refreshTokens,
		tx,
		usecase.NewPasswordHasher(),
		tokens,
		testutil.FixedClock{Current: time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)},
	)
	userController := NewUserController(userUsecase)

	e := echo.New()
	e.POST("/api/auth/signup", userController.SignUp)
	e.POST("/api/auth/login", userController.Login)

	signupBody := `{"name":"API Test User","email":"api-integration@example.com","password":"StrongPassword123!"}`
	signup := performJSONRequest(e, http.MethodPost, "/api/auth/signup", signupBody)
	if signup.Code != http.StatusCreated {
		t.Fatalf("signup status=%d body=%s", signup.Code, signup.Body.String())
	}
	var signupPayload SignUpResponse
	if err := json.Unmarshal(signup.Body.Bytes(), &signupPayload); err != nil {
		t.Fatalf("decode signup response: %v", err)
	}
	if signupPayload.AccessToken == "" || signupPayload.User.Email != "api-integration@example.com" {
		t.Fatalf("unexpected signup response: %+v", signupPayload)
	}
	if len(signup.Result().Cookies()) < 2 {
		t.Fatalf("signup must issue auth cookies; got %d", len(signup.Result().Cookies()))
	}

	var userCount int64
	if err := db.Table("users").Where("email = ?", "api-integration@example.com").Count(&userCount).Error; err != nil {
		t.Fatalf("count created user: %v", err)
	}
	if userCount != 1 {
		t.Fatalf("persisted user count=%d want=1", userCount)
	}

	duplicate := performJSONRequest(e, http.MethodPost, "/api/auth/signup", signupBody)
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate signup status=%d body=%s", duplicate.Code, duplicate.Body.String())
	}

	login := performJSONRequest(e, http.MethodPost, "/api/auth/login", `{"email":"api-integration@example.com","password":"StrongPassword123!"}`)
	if login.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", login.Code, login.Body.String())
	}
	var loginPayload LoginResponse
	if err := json.Unmarshal(login.Body.Bytes(), &loginPayload); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if loginPayload.AccessToken == "" || loginPayload.User.Email != "api-integration@example.com" {
		t.Fatalf("unexpected login response: %+v", loginPayload)
	}
}

func performJSONRequest(e *echo.Echo, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, req)
	return recorder
}

func TestAuthAPIIntegrationLogoutInvalidatesRefreshAndAccessTokens(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-jwt-secret-at-least-32-characters-long")
	t.Setenv("APP_ENV", "test")
	t.Setenv("COOKIE_SAME_SITE", "strict")

	db := testutil.OpenTestDB(t)
	if err := migrate.Run(db, migrate.Config{AppEnvironment: "test"}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	testutil.ResetDatabase(t, db)
	t.Cleanup(func() { testutil.ResetDatabase(t, db) })

	users := repository.NewUserRepository(db)
	refreshTokens := repository.NewRefreshTokenRepository(db)
	tx := repository.NewAuthTransactionManager(db)
	tokens, err := usecase.NewTokenServiceFromEnv()
	if err != nil {
		t.Fatalf("create token service: %v", err)
	}
	userUsecase := usecase.NewUserUsecase(
		users,
		validator.NewUserValidator(),
		refreshTokens,
		tx,
		usecase.NewPasswordHasher(),
		tokens,
		usecase.NewClock(),
	)
	userController := NewUserController(userUsecase)
	authMiddleware := appMiddleware.NewAuthMiddleware(tokens, userUsecase)
	csrf := appMiddleware.NewCSRFMiddleware("csrf_token")

	e := echo.New()
	e.POST("/api/auth/signup", userController.SignUp)
	e.POST("/api/auth/logout", userController.Logout, csrf)
	e.POST("/api/auth/refresh", userController.Refresh, csrf)
	e.GET("/api/users/me", userController.CurrentUser, authMiddleware.Authenticate)

	signupBody := `{"name":"Logout Integration User","email":"logout-integration@example.com","password":"StrongPassword123!"}`
	signup := performJSONRequest(e, http.MethodPost, "/api/auth/signup", signupBody)
	if signup.Code != http.StatusCreated {
		t.Fatalf("signup status=%d body=%s", signup.Code, signup.Body.String())
	}

	var signupPayload SignUpResponse
	if err := json.Unmarshal(signup.Body.Bytes(), &signupPayload); err != nil {
		t.Fatalf("decode signup response: %v", err)
	}
	if signupPayload.AccessToken == "" {
		t.Fatal("signup did not return an access token")
	}

	var refreshCookie, csrfCookie *http.Cookie
	for _, cookie := range signup.Result().Cookies() {
		switch cookie.Name {
		case "refresh_token":
			refreshCookie = cookie
		case "csrf_token":
			csrfCookie = cookie
		}
	}
	if refreshCookie == nil || refreshCookie.Value == "" {
		t.Fatal("signup did not issue refresh_token cookie")
	}
	if csrfCookie == nil || csrfCookie.Value == "" {
		t.Fatal("signup did not issue csrf_token cookie")
	}

	refreshHash := tokens.HashRefreshToken(refreshCookie.Value)
	storedRefresh, err := refreshTokens.FindByTokenHash(t.Context(), refreshHash)
	if err != nil {
		t.Fatalf("find issued refresh token: %v", err)
	}
	familyID := storedRefresh.FamilyID

	var beforeLogout struct {
		TokenVersion uint64 `gorm:"column:token_version"`
	}
	if err := db.Table("users").Select("token_version").Where("id = ?", signupPayload.User.ID).Scan(&beforeLogout).Error; err != nil {
		t.Fatalf("read token_version before logout: %v", err)
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	logoutReq.AddCookie(refreshCookie)
	logoutReq.AddCookie(csrfCookie)
	logoutReq.Header.Set("X-CSRF-Token", csrfCookie.Value)
	logoutRec := httptest.NewRecorder()
	e.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusNoContent {
		t.Fatalf("logout status=%d body=%s", logoutRec.Code, logoutRec.Body.String())
	}

	var afterLogout struct {
		TokenVersion uint64 `gorm:"column:token_version"`
	}
	if err := db.Table("users").Select("token_version").Where("id = ?", signupPayload.User.ID).Scan(&afterLogout).Error; err != nil {
		t.Fatalf("read token_version after logout: %v", err)
	}
	if afterLogout.TokenVersion != beforeLogout.TokenVersion+1 {
		t.Fatalf("token_version after logout=%d want=%d", afterLogout.TokenVersion, beforeLogout.TokenVersion+1)
	}

	var activeFamilyTokens int64
	if err := db.Table("refresh_tokens").Where("family_id = ? AND revoked_at IS NULL", familyID).Count(&activeFamilyTokens).Error; err != nil {
		t.Fatalf("count active family tokens: %v", err)
	}
	if activeFamilyTokens != 0 {
		t.Fatalf("active refresh tokens in logged-out family=%d want=0", activeFamilyTokens)
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	refreshReq.AddCookie(refreshCookie)
	refreshReq.AddCookie(csrfCookie)
	refreshReq.Header.Set("X-CSRF-Token", csrfCookie.Value)
	refreshRec := httptest.NewRecorder()
	e.ServeHTTP(refreshRec, refreshReq)
	if refreshRec.Code != http.StatusUnauthorized {
		t.Fatalf("old refresh token after logout status=%d want=401 body=%s", refreshRec.Code, refreshRec.Body.String())
	}
	var refreshError apiErrorResponse
	if err := json.Unmarshal(refreshRec.Body.Bytes(), &refreshError); err != nil {
		t.Fatalf("decode refresh error: %v", err)
	}
	if refreshError.Code != "INVALID_REFRESH_TOKEN" {
		t.Fatalf("old refresh token error code=%q want=INVALID_REFRESH_TOKEN", refreshError.Code)
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	meReq.Header.Set(echo.HeaderAuthorization, "Bearer "+signupPayload.AccessToken)
	meRec := httptest.NewRecorder()
	e.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("old access token after logout status=%d want=401 body=%s", meRec.Code, meRec.Body.String())
	}
}
