package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type IRoleMiddleware interface {
	Authorize(next echo.HandlerFunc) echo.HandlerFunc
}

type roleMiddleware struct {
	allowed map[entity.Role]struct{}
	logger  *slog.Logger
}

func NewRoleMiddleware(logger *slog.Logger, allowed ...entity.Role) (IRoleMiddleware, error) {
	if logger == nil || len(allowed) == 0 {
		return nil, entity.ErrInvalidRoleConfiguration
	}
	roles := make(map[entity.Role]struct{}, len(allowed))
	for _, role := range allowed {
		if !role.IsPersistable() {
			return nil, entity.ErrInvalidRoleConfiguration
		}
		roles[role] = struct{}{}
	}
	return &roleMiddleware{allowed: roles, logger: logger}, nil
}

func (m *roleMiddleware) Authorize(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID, ok := c.Get(AuthUserIDKey).(uuid.UUID)
		if !ok || userID == uuid.Nil {
			return echo.NewHTTPError(http.StatusUnauthorized)
		}
		role, ok := c.Get(AuthUserRoleKey).(entity.Role)
		if !ok || !role.IsPersistable() {
			return echo.NewHTTPError(http.StatusUnauthorized)
		}
		if _, allowed := m.allowed[role]; !allowed {
			m.logger.Warn("role access denied", "user_id", userID, "role", role, "path", c.Path(), "method", c.Request().Method, "occurred_at", time.Now().UTC())
			return echo.NewHTTPError(http.StatusForbidden)
		}
		return next(c)
	}
}
