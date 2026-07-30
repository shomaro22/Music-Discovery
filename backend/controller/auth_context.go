package controller

import (
	"music-discovery/backend/middleware"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func authUserID(ctx echo.Context) (uuid.UUID, bool) {
	value, ok := ctx.Get(middleware.AuthUserIDKey).(uuid.UUID)
	return value, ok && value != uuid.Nil
}
