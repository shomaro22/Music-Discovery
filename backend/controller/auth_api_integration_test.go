package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
