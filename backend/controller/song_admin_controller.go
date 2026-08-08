package controller

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"music-discovery/backend/entity"
	"music-discovery/backend/usecase"
)

type ISongAdminController interface {
	ListSongs(echo.Context) error
	GetSong(echo.Context) error
	CreateSong(echo.Context) error
	UpdateSong(echo.Context) error
	DeleteSong(echo.Context) error
}

type songAdminController struct {
	usecase usecase.IContentAdminUsecase
}

type songTagAdminRequest struct {
	TagID  uuid.UUID `json:"tag_id"`
	Weight float64   `json:"weight"`
}

type songAdminRequest struct {
	ArtistID       uuid.UUID             `json:"artist_id"`
	SpotifyTrackID string                `json:"spotify_track_id"`
	Name           string                `json:"name"`
	AlbumName      string                `json:"album_name"`
	ImageURL       string                `json:"image_url"`
	SpotifyURL     string                `json:"spotify_url"`
	PreviewURL     string                `json:"preview_url"`
	DurationMs     int                   `json:"duration_ms"`
	Popularity     int                   `json:"popularity"`
	ReleaseDate    string                `json:"release_date"`
	Explicit       bool                  `json:"explicit"`
	Tags           []songTagAdminRequest `json:"tags"`
}

func NewSongAdminController(value usecase.IContentAdminUsecase) ISongAdminController {
	return &songAdminController{usecase: value}
}

func (r songAdminRequest) toInput(id uuid.UUID) (usecase.SongInput, error) {
	var releaseDate time.Time
	if strings.TrimSpace(r.ReleaseDate) != "" {
		value, err := time.Parse("2006-01-02", r.ReleaseDate)
		if err != nil {
			return usecase.SongInput{}, entity.ErrInvalidInput
		}
		releaseDate = value
	}

	tags := make([]usecase.SongTagInput, 0, len(r.Tags))
	for _, tag := range r.Tags {
		tags = append(tags, usecase.SongTagInput{TagID: tag.TagID, Weight: tag.Weight})
	}

	return usecase.SongInput{
		Song: entity.Song{
			ID:             id,
			ArtistID:       r.ArtistID,
			SpotifyTrackID: r.SpotifyTrackID,
			Name:           r.Name,
			AlbumName:      r.AlbumName,
			ImageURL:       r.ImageURL,
			SpotifyURL:     r.SpotifyURL,
			PreviewURL:     r.PreviewURL,
			DurationMs:     r.DurationMs,
			Popularity:     r.Popularity,
			ReleaseDate:    releaseDate,
			Explicit:       r.Explicit,
		},
		Tags: tags,
	}, nil
}

func (c *songAdminController) ListSongs(ctx echo.Context) error {
	input, err := adminPageInput(ctx)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	value, err := c.usecase.ListSongs(ctx.Request().Context(), input)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, mapAdminPage(value, toSongAdminResponse))
}

func (c *songAdminController) GetSong(ctx echo.Context) error {
	id, err := adminPathID(ctx, "songId")
	if err != nil {
		return adminResponseError(ctx, entity.ErrInvalidInput)
	}
	value, err := c.usecase.GetSong(ctx.Request().Context(), id)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, toSongAdminResponse(*value))
}

func (c *songAdminController) CreateSong(ctx echo.Context) error {
	return c.saveSong(ctx, uuid.Nil, http.StatusCreated)
}

func (c *songAdminController) UpdateSong(ctx echo.Context) error {
	id, err := adminPathID(ctx, "songId")
	if err != nil {
		return adminResponseError(ctx, entity.ErrInvalidInput)
	}
	return c.saveSong(ctx, id, http.StatusOK)
}

func (c *songAdminController) saveSong(ctx echo.Context, id uuid.UUID, status int) error {
	actorID, err := adminActorID(ctx)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	var request songAdminRequest
	if err := ctx.Bind(&request); err != nil {
		return adminResponseError(ctx, entity.ErrInvalidInput)
	}
	input, err := request.toInput(id)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	value, err := c.usecase.SaveSong(ctx.Request().Context(), actorID, input, adminAuditInput(ctx))
	if err != nil {
		return adminResponseError(ctx, err)
	}
	return ctx.JSON(status, toSongAdminResponse(*value))
}

func (c *songAdminController) DeleteSong(ctx echo.Context) error {
	actorID, err := adminActorID(ctx)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	id, err := adminPathID(ctx, "songId")
	if err != nil {
		return adminResponseError(ctx, entity.ErrInvalidInput)
	}
	if err := c.usecase.DeleteSong(ctx.Request().Context(), actorID, id, adminAuditInput(ctx)); err != nil {
		return adminResponseError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}
