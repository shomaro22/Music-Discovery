package middleware

import (
	"context"
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

type authMiddlewareTokenService struct {
	userID       uuid.UUID
	tokenVersion uint64
	verifyErr    error
}

func (f *authMiddlewareTokenService) ComparePassword(string, string) error { return nil }
func (f *authMiddlewareTokenService) GenerateAccessToken(*entity.User, time.Time) (string, int64, error) {
	return "", 0, nil
}
func (f *authMiddlewareTokenService) GenerateRefreshToken() (string, string, error) {
	return "", "", nil
}
func (f *authMiddlewareTokenService) HashRefreshToken(string) string     { return "" }
func (f *authMiddlewareTokenService) GenerateCSRFToken() (string, error) { return "", nil }
func (f *authMiddlewareTokenService) GenerateFamilyID() uuid.UUID        { return uuid.New() }
func (f *authMiddlewareTokenService) RefreshTokenTTL() time.Duration     { return time.Hour }
func (f *authMiddlewareTokenService) VerifyAccessToken(string, time.Time) (uuid.UUID, uint64, error) {
	return f.userID, f.tokenVersion, f.verifyErr
}

type authMiddlewareUserUsecase struct {
	user *entity.User
	err  error
}

func (f *authMiddlewareUserUsecase) SignUp(context.Context, usecase.SignUpInput) (*entity.User, error) {
	return nil, nil
}
func (f *authMiddlewareUserUsecase) Login(context.Context, usecase.LoginInput) (*usecase.LoginResult, error) {
	return nil, nil
}
func (f *authMiddlewareUserUsecase) Refresh(context.Context, string) (*usecase.RefreshResult, error) {
	return nil, nil
}
func (f *authMiddlewareUserUsecase) ValidateTokenVersion(context.Context, uuid.UUID, uint64) (*entity.User, error) {
	return f.user, f.err
}

func TestAuthMiddlewareAcceptsValidTokenVersion(t *testing.T) {
	userID := uuid.New()
	user := &entity.User{ID: userID, Status: entity.UserStatusActive, TokenVersion: 2}
	middleware := NewAuthMiddleware(&authMiddlewareTokenService{userID: userID, tokenVersion: 2}, &authMiddlewareUserUsecase{user: user})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer valid")
	rec := httptest.NewRecorder()
	err := middleware.Authenticate(func(c echo.Context) error {
		if c.Get("auth_user") != user {
			t.Fatal("auth_user was not stored")
		}
		return c.NoContent(http.StatusNoContent)
	})(e.NewContext(req, rec))
	if err != nil || rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d err=%v", rec.Code, err)
	}
}

func TestAuthMiddlewareRejectsInvalidSignatureOrExpiry(t *testing.T) {
	middleware := NewAuthMiddleware(&authMiddlewareTokenService{verifyErr: entity.ErrInvalidCredentials}, &authMiddlewareUserUsecase{})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer invalid")
	err := middleware.Authenticate(func(c echo.Context) error { return nil })(e.NewContext(req, httptest.NewRecorder()))
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusUnauthorized {
		t.Fatalf("err=%v", err)
	}
}

func TestAuthMiddlewareRejectsTokenVersionMismatch(t *testing.T) {
	middleware := NewAuthMiddleware(&authMiddlewareTokenService{userID: uuid.New(), tokenVersion: 1}, &authMiddlewareUserUsecase{err: errors.New("token version mismatch")})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer stale")
	err := middleware.Authenticate(func(c echo.Context) error { return nil })(e.NewContext(req, httptest.NewRecorder()))
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusUnauthorized {
		t.Fatalf("err=%v", err)
	}
}
