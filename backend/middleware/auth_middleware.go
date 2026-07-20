package middleware

import (
	"github.com/labstack/echo/v4"
	"music-discovery/backend/entity"
	"music-discovery/backend/usecase"
	"net/http"
	"strings"
	"time"
)

type IAuthMiddleware interface {
	Authenticate(echo.HandlerFunc) echo.HandlerFunc
}
type authMiddleware struct {
	tokenService usecase.ITokenService
	userUsecase  usecase.IUserUsecase
}

func NewAuthMiddleware(tokenService usecase.ITokenService, userUsecase usecase.IUserUsecase) IAuthMiddleware {
	return &authMiddleware{tokenService: tokenService, userUsecase: userUsecase}
}
func (m *authMiddleware) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authorization := c.Request().Header.Get(echo.HeaderAuthorization)
		if !strings.HasPrefix(authorization, "Bearer ") {
			return echo.NewHTTPError(http.StatusUnauthorized)
		}
		userID, tokenVersion, err := m.tokenService.VerifyAccessToken(strings.TrimPrefix(authorization, "Bearer "), time.Now().UTC())
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized)
		}
		user, err := m.userUsecase.ValidateTokenVersion(c.Request().Context(), userID, tokenVersion)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, entity.ErrInvalidCredentials)
		}
		c.Set("auth_user", user)
		return next(c)
	}
}
