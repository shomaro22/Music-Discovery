package usecase

import (
	"context"
	"strings"

	"music-discovery/backend/entity"
	"music-discovery/backend/spotify"
)

type ISpotifyAdminUsecase interface {
	SearchArtists(context.Context, string, int) (spotify.SearchArtistsResult, error)
	SearchTracks(context.Context, string, int) (spotify.SearchTracksResult, error)
}

type spotifyAdminUsecase struct{ client spotify.IClient }

func NewSpotifyAdminUsecase(client spotify.IClient) ISpotifyAdminUsecase {
	return &spotifyAdminUsecase{client: client}
}

func (u *spotifyAdminUsecase) SearchArtists(ctx context.Context, query string, limit int) (spotify.SearchArtistsResult, error) {
	if strings.TrimSpace(query) == "" || limit < 1 || limit > 10 {
		return spotify.SearchArtistsResult{}, entity.ErrInvalidInput
	}
	value, err := u.client.SearchArtists(ctx, query, limit)
	if err != nil {
		return spotify.SearchArtistsResult{}, entity.ErrInternal
	}
	return value, nil
}

func (u *spotifyAdminUsecase) SearchTracks(ctx context.Context, query string, limit int) (spotify.SearchTracksResult, error) {
	if strings.TrimSpace(query) == "" || limit < 1 || limit > 10 {
		return spotify.SearchTracksResult{}, entity.ErrInvalidInput
	}
	value, err := u.client.SearchTracks(ctx, query, limit)
	if err != nil {
		return spotify.SearchTracksResult{}, entity.ErrInternal
	}
	return value, nil
}
