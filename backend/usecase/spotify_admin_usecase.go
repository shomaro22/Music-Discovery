package usecase

import (
	"context"
	"errors"
	"strings"

	"music-discovery/backend/entity"
	"music-discovery/backend/spotify"
)

type SpotifyArtistPreview struct {
	SpotifyArtistID string
	Name            string
	Genres          []string
	ImageURL        string
	SpotifyURL      string
	Popularity      int
}

type SpotifyArtistSearchResult struct {
	Items []SpotifyArtistPreview
	Total int
}

type SpotifyTrackArtistPreview struct {
	SpotifyArtistID string
	Name            string
}

type SpotifyTrackPreview struct {
	SpotifyTrackID string
	Name           string
	Artists        []SpotifyTrackArtistPreview
	AlbumName      string
	ReleaseDate    string
	ImageURL       string
	DurationMS     int
	Explicit       bool
	Popularity     int
	PreviewURL     string
	SpotifyURL     string
}

type SpotifyTrackSearchResult struct {
	Items []SpotifyTrackPreview
	Total int
}

type ISpotifyAdminUsecase interface {
	SearchArtists(context.Context, string, int) (SpotifyArtistSearchResult, error)
	SearchTracks(context.Context, string, int) (SpotifyTrackSearchResult, error)
}

type spotifyAdminUsecase struct{ client spotify.IClient }

func NewSpotifyAdminUsecase(client spotify.IClient) ISpotifyAdminUsecase {
	return &spotifyAdminUsecase{client: client}
}

func (u *spotifyAdminUsecase) SearchArtists(ctx context.Context, query string, limit int) (SpotifyArtistSearchResult, error) {
	if strings.TrimSpace(query) == "" || limit < 1 || limit > 10 {
		return SpotifyArtistSearchResult{}, entity.ErrInvalidInput
	}
	value, err := u.client.SearchArtists(ctx, query, limit)
	if err != nil {
		return SpotifyArtistSearchResult{}, spotifyUsecaseError(err)
	}

	items := make([]SpotifyArtistPreview, 0, len(value.Items))
	for _, artist := range value.Items {
		items = append(items, SpotifyArtistPreview{
			SpotifyArtistID: artist.ID,
			Name:            artist.Name,
			Genres:          append([]string(nil), artist.Genres...),
			ImageURL:        firstSpotifyImageURL(artist.Images),
			SpotifyURL:      artist.ExternalURLs["spotify"],
			Popularity:      artist.Popularity,
		})
	}
	return SpotifyArtistSearchResult{Items: items, Total: value.Total}, nil
}

func (u *spotifyAdminUsecase) SearchTracks(ctx context.Context, query string, limit int) (SpotifyTrackSearchResult, error) {
	if strings.TrimSpace(query) == "" || limit < 1 || limit > 10 {
		return SpotifyTrackSearchResult{}, entity.ErrInvalidInput
	}
	value, err := u.client.SearchTracks(ctx, query, limit)
	if err != nil {
		return SpotifyTrackSearchResult{}, spotifyUsecaseError(err)
	}

	items := make([]SpotifyTrackPreview, 0, len(value.Items))
	for _, track := range value.Items {
		artists := make([]SpotifyTrackArtistPreview, 0, len(track.Artists))
		for _, artist := range track.Artists {
			artists = append(artists, SpotifyTrackArtistPreview{
				SpotifyArtistID: artist.ID,
				Name:            artist.Name,
			})
		}
		items = append(items, SpotifyTrackPreview{
			SpotifyTrackID: track.ID,
			Name:           track.Name,
			Artists:        artists,
			AlbumName:      track.Album.Name,
			ReleaseDate:    track.Album.ReleaseDate,
			ImageURL:       firstSpotifyImageURL(track.Album.Images),
			DurationMS:     track.DurationMS,
			Explicit:       track.Explicit,
			Popularity:     track.Popularity,
			PreviewURL:     track.PreviewURL,
			SpotifyURL:     track.ExternalURLs["spotify"],
		})
	}
	return SpotifyTrackSearchResult{Items: items, Total: value.Total}, nil
}

func spotifyUsecaseError(err error) error {
	if errors.Is(err, spotify.ErrUnavailable) {
		return entity.ErrSpotifyUnavailable
	}
	return entity.ErrInternal
}

func firstSpotifyImageURL(images []spotify.Image) string {
	if len(images) == 0 {
		return ""
	}
	return images[0].URL
}
