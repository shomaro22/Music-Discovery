package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type IAuthMiddleware interface {
	Authenticate(echo.HandlerFunc) echo.HandlerFunc
}

type ITokenVersionValidator interface {
	ValidateTokenVersion(
		context.Context,
		uuid.UUID,
		uint64,
	) (*entity.User, error)
}

type authMiddleware struct {
	tokenService          usecase.ITokenService
	tokenVersionValidator ITokenVersionValidator
}

func NewAuthMiddleware(
	tokenService usecase.ITokenService,
	tokenVersionValidator ITokenVersionValidator,
) IAuthMiddleware {
	return &authMiddleware{
		tokenService:          tokenService,
		tokenVersionValidator: tokenVersionValidator,
	}
}

func (m *authMiddleware) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authorization := c.Request().Header.Get(echo.HeaderAuthorization)
		if !strings.HasPrefix(authorization, "Bearer ") {
			return echo.NewHTTPError(http.StatusUnauthorized)
		}

		userID, tokenVersion, err := m.tokenService.VerifyAccessToken(
			strings.TrimPrefix(authorization, "Bearer "),
			time.Now().UTC(),
		)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized)
		}

		user, err := m.tokenVersionValidator.ValidateTokenVersion(
			c.Request().Context(),
			userID,
			tokenVersion,
		)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, entity.ErrInvalidCredentials)
		}

		c.Set("auth_user", user)
		return next(c)
	}
}
