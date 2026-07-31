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

type SpotifyArtistPreviewResponse struct {
	SpotifyArtistID string   `json:"spotify_artist_id"`
	Name            string   `json:"name"`
	Genres          []string `json:"genres"`
	ImageURL        string   `json:"image_url"`
	SpotifyURL      string   `json:"spotify_url"`
	Popularity      int      `json:"popularity"`
}

type SpotifyArtistSearchResponse struct {
	Items []SpotifyArtistPreviewResponse `json:"items"`
	Total int                            `json:"total"`
}

type SpotifyTrackArtistPreviewResponse struct {
	SpotifyArtistID string `json:"spotify_artist_id"`
	Name            string `json:"name"`
}

type SpotifyTrackPreviewResponse struct {
	SpotifyTrackID string                              `json:"spotify_track_id"`
	Name           string                              `json:"name"`
	Artists        []SpotifyTrackArtistPreviewResponse `json:"artists"`
	AlbumName      string                              `json:"album_name"`
	ReleaseDate    string                              `json:"release_date"`
	ImageURL       string                              `json:"image_url"`
	DurationMS     int                                 `json:"duration_ms"`
	Explicit       bool                                `json:"explicit"`
	Popularity     int                                 `json:"popularity"`
	PreviewURL     string                              `json:"preview_url"`
	SpotifyURL     string                              `json:"spotify_url"`
}

type SpotifyTrackSearchResponse struct {
	Items []SpotifyTrackPreviewResponse `json:"items"`
	Total int                           `json:"total"`
}

func NewSpotifyAdminController(value usecase.ISpotifyAdminUsecase) ISpotifyAdminController {
	return &spotifyAdminController{usecase: value}
}

func spotifySearchInput(ctx echo.Context) (string, int, error) {
	query := strings.TrimSpace(ctx.QueryParam("q"))
	limit := 10
	if raw := strings.TrimSpace(ctx.QueryParam("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return "", 0, entity.ErrInvalidInput
		}
		limit = value
	}
	if query == "" || limit < 1 || limit > 10 {
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
	return adminJSON(ctx, http.StatusOK, spotifyArtistSearchResponse(value), err)
}

func (c *spotifyAdminController) SearchTracks(ctx echo.Context) error {
	query, limit, err := spotifySearchInput(ctx)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	value, err := c.usecase.SearchTracks(ctx.Request().Context(), query, limit)
	return adminJSON(ctx, http.StatusOK, spotifyTrackSearchResponse(value), err)
}

func spotifyArtistSearchResponse(value usecase.SpotifyArtistSearchResult) SpotifyArtistSearchResponse {
	items := make([]SpotifyArtistPreviewResponse, 0, len(value.Items))
	for _, item := range value.Items {
		items = append(items, SpotifyArtistPreviewResponse{
			SpotifyArtistID: item.SpotifyArtistID,
			Name:            item.Name,
			Genres:          append([]string(nil), item.Genres...),
			ImageURL:        item.ImageURL,
			SpotifyURL:      item.SpotifyURL,
			Popularity:      item.Popularity,
		})
	}
	return SpotifyArtistSearchResponse{Items: items, Total: value.Total}
}

func spotifyTrackSearchResponse(value usecase.SpotifyTrackSearchResult) SpotifyTrackSearchResponse {
	items := make([]SpotifyTrackPreviewResponse, 0, len(value.Items))
	for _, item := range value.Items {
		artists := make([]SpotifyTrackArtistPreviewResponse, 0, len(item.Artists))
		for _, artist := range item.Artists {
			artists = append(artists, SpotifyTrackArtistPreviewResponse{
				SpotifyArtistID: artist.SpotifyArtistID,
				Name:            artist.Name,
			})
		}
		items = append(items, SpotifyTrackPreviewResponse{
			SpotifyTrackID: item.SpotifyTrackID,
			Name:           item.Name,
			Artists:        artists,
			AlbumName:      item.AlbumName,
			ReleaseDate:    item.ReleaseDate,
			ImageURL:       item.ImageURL,
			DurationMS:     item.DurationMS,
			Explicit:       item.Explicit,
			Popularity:     item.Popularity,
			PreviewURL:     item.PreviewURL,
			SpotifyURL:     item.SpotifyURL,
		})
	}
	return SpotifyTrackSearchResponse{Items: items, Total: value.Total}
}
