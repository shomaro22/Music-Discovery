package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"music-discovery/backend/entity"

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

type authMiddlewareTokenVersionValidator struct {
	user *entity.User
	err  error
}

func (f *authMiddlewareTokenVersionValidator) ValidateTokenVersion(
	context.Context,
	uuid.UUID,
	uint64,
) (*entity.User, error) {
	return f.user, f.err
}

func TestAuthMiddlewareAcceptsValidTokenVersion(t *testing.T) {
	userID := uuid.New()
	user := &entity.User{ID: userID, Status: entity.UserStatusActive, TokenVersion: 2}
	middleware := NewAuthMiddleware(&authMiddlewareTokenService{userID: userID, tokenVersion: 2}, &authMiddlewareTokenVersionValidator{user: user})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer valid")
	rec := httptest.NewRecorder()
	err := middleware.Authenticate(func(c echo.Context) error {
		if got := c.Get("auth_user_id"); got != user.ID {
			t.Fatalf("auth_user_id=%v want=%v", got, user.ID)
		}
		if got := c.Get("auth_user_role"); got != user.Role {
			t.Fatalf("auth_user_role=%v want=%v", got, user.Role)
		}
		if got := c.Get("auth_user"); got != nil {
			t.Fatalf("auth_user must not contain the User entity: %v", got)
		}
		return c.NoContent(http.StatusNoContent)
	})(e.NewContext(req, rec))
	if err != nil || rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d err=%v", rec.Code, err)
	}
}

func TestAuthMiddlewareRejectsInvalidSignatureOrExpiry(t *testing.T) {
	middleware := NewAuthMiddleware(&authMiddlewareTokenService{verifyErr: entity.ErrInvalidCredentials}, &authMiddlewareTokenVersionValidator{})
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
	middleware := NewAuthMiddleware(&authMiddlewareTokenService{userID: uuid.New(), tokenVersion: 1}, &authMiddlewareTokenVersionValidator{err: entity.ErrInvalidCredentials})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer stale")
	err := middleware.Authenticate(func(c echo.Context) error { return nil })(e.NewContext(req, httptest.NewRecorder()))
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusUnauthorized {
		t.Fatalf("err=%v", err)
	}
}
