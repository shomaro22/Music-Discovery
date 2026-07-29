package controller

import (
	"net/http"
	"strconv"
	"strings"

	"music-discovery/backend/entity"
	"music-discovery/backend/usecase"

	"github.com/labstack/echo/v4"
)

type ISpotifyAdminController interface {
	SearchArtists(echo.Context) error
	SearchTracks(echo.Context) error
}

type spotifyAdminController struct{ usecase usecase.ISpotifyAdminUsecase }

func NewSpotifyAdminController(value usecase.ISpotifyAdminUsecase) ISpotifyAdminController {
	return &spotifyAdminController{usecase: value}
}

func spotifySearchInput(ctx echo.Context) (string, int, error) {
	query := strings.TrimSpace(ctx.QueryParam("q"))
	limit := 20
	if raw := strings.TrimSpace(ctx.QueryParam("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return "", 0, entity.ErrInvalidInput
		}
		limit = value
	}
	if query == "" || limit < 1 || limit > 50 {
		return "", 0, entity.ErrInvalidInput
	}
	return query, limit, nil
}

func (c *spotifyAdminController) SearchArtists(ctx echo.Context) error {
	query, limit, err := spotifySearchInput(ctx)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	value, err := c.usecase.SearchArtists(ctx.Request().Context(), query, limit)
	return adminJSON(ctx, http.StatusOK, value, err)
}

func (c *spotifyAdminController) SearchTracks(ctx echo.Context) error {
	query, limit, err := spotifySearchInput(ctx)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	value, err := c.usecase.SearchTracks(ctx.Request().Context(), query, limit)
	return adminJSON(ctx, http.StatusOK, value, err)
}
