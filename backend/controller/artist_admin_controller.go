package controller

import (
	"net/http"

	"music-discovery/backend/entity"
	"music-discovery/backend/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type IArtistAdminController interface {
	ListArtists(echo.Context) error
	GetArtist(echo.Context) error
	CreateArtist(echo.Context) error
	UpdateArtist(echo.Context) error
	DeleteArtist(echo.Context) error
}

type artistAdminController struct {
	usecase usecase.IContentAdminUsecase
}

type artistAdminRequest struct {
	SpotifyArtistID string      `json:"spotify_artist_id"`
	Name            string      `json:"name"`
	ImageURL        string      `json:"image_url"`
	SpotifyURL      string      `json:"spotify_url"`
	Popularity      int         `json:"popularity"`
	TagIDs          []uuid.UUID `json:"tag_ids"`
}

func NewArtistAdminController(value usecase.IContentAdminUsecase) IArtistAdminController {
	return &artistAdminController{usecase: value}
}

func (r artistAdminRequest) toInput(id uuid.UUID) usecase.ArtistInput {
	return usecase.ArtistInput{
		ID:              id,
		SpotifyArtistID: r.SpotifyArtistID,
		Name:            r.Name,
		ImageURL:        r.ImageURL,
		SpotifyURL:      r.SpotifyURL,
		Popularity:      r.Popularity,
		TagIDs:          r.TagIDs,
	}
}

func (c *artistAdminController) ListArtists(ctx echo.Context) error {
	input, err := adminPageInput(ctx)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	value, err := c.usecase.ListArtists(ctx.Request().Context(), input)
	return adminJSON(ctx, http.StatusOK, value, err)
}

func (c *artistAdminController) GetArtist(ctx echo.Context) error {
	id, err := adminPathID(ctx, "artistId")
	if err != nil {
		return adminResponseError(ctx, entity.ErrInvalidInput)
	}
	value, err := c.usecase.GetArtist(ctx.Request().Context(), id)
	return adminJSON(ctx, http.StatusOK, value, err)
}

func (c *artistAdminController) CreateArtist(ctx echo.Context) error {
	return c.saveArtist(ctx, uuid.Nil, http.StatusCreated)
}

func (c *artistAdminController) UpdateArtist(ctx echo.Context) error {
	id, err := adminPathID(ctx, "artistId")
	if err != nil {
		return adminResponseError(ctx, entity.ErrInvalidInput)
	}
	return c.saveArtist(ctx, id, http.StatusOK)
}

func (c *artistAdminController) saveArtist(ctx echo.Context, id uuid.UUID, status int) error {
	actorID, err := adminActorID(ctx)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	var request artistAdminRequest
	if err := bindAdminRequest(ctx, &request); err != nil {
		return adminResponseError(ctx, err)
	}
	value, err := c.usecase.SaveArtist(ctx.Request().Context(), actorID, request.toInput(id), adminAuditInput(ctx))
	return adminJSON(ctx, status, value, err)
}

func (c *artistAdminController) DeleteArtist(ctx echo.Context) error {
	actorID, id, err := adminActorAndPathID(ctx, "artistId")
	if err != nil {
		return adminResponseError(ctx, err)
	}
	err = c.usecase.DeleteArtist(ctx.Request().Context(), actorID, id, adminAuditInput(ctx))
	return adminNoContent(ctx, err)
}

func adminActorAndPathID(ctx echo.Context, pathName string) (uuid.UUID, uuid.UUID, error) {
	actorID, err := adminActorID(ctx)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	id, err := adminPathID(ctx, pathName)
	if err != nil {
		return uuid.Nil, uuid.Nil, entity.ErrInvalidInput
	}
	return actorID, id, nil
}

func adminJSON(ctx echo.Context, status int, value interface{}, err error) error {
	if err != nil {
		return adminResponseError(ctx, err)
	}
	return ctx.JSON(status, value)
}

func adminNoContent(ctx echo.Context, err error) error {
	if err != nil {
		return adminResponseError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}
