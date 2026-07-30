package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/repository"
	"music-discovery/backend/spotify"
)

type SpotifyCandidateImportConfig struct {
	SearchLimit int
	MaxQueries  int
}

type ISpotifyCandidateImportUsecase interface {
	Import(context.Context, []string) (int, error)
}

type spotifyCandidateImportUsecase struct {
	client     spotify.IClient
	repository repository.ISpotifyCandidateRepository
	config     SpotifyCandidateImportConfig
}

func NewSpotifyCandidateImportUsecase(client spotify.IClient, repository repository.ISpotifyCandidateRepository, config SpotifyCandidateImportConfig) ISpotifyCandidateImportUsecase {
	return &spotifyCandidateImportUsecase{client: client, repository: repository, config: config}
}

func (u *spotifyCandidateImportUsecase) Import(ctx context.Context, genres []string) (int, error) {
	queries := spotifyCandidateQueries(genres, u.config.MaxQueries)
	limit := u.config.SearchLimit
	if limit < 1 || limit > 10 {
		limit = 10
	}

	byTrackID := make(map[string]entity.SpotifyCandidate)
	for _, query := range queries {
		result, err := u.client.SearchTracks(ctx, query, limit)
		if err != nil {
			return 0, entity.ErrSpotifyCandidateImportFailed
		}
		for _, track := range result.Items {
			candidate, ok := spotifyTrackCandidate(track)
			if ok {
				byTrackID[track.ID] = candidate
			}
		}
	}

	values := make([]entity.SpotifyCandidate, 0, len(byTrackID))
	for _, value := range byTrackID {
		values = append(values, value)
	}
	if err := u.repository.UpsertCandidates(ctx, values); err != nil {
		return 0, entity.ErrSpotifyCandidateImportFailed
	}
	return len(values), nil
}

func spotifyCandidateQueries(genres []string, maxQueries int) []string {
	if maxQueries <= 0 {
		maxQueries = 3
	}
	result := make([]string, 0, maxQueries)
	seen := map[string]bool{}
	for _, rawGenre := range genres {
		// GENRE_POP → pop
		// GENRE_HIP_HOP → hip-hop
		// GENRE_K_POP → k-pop
		genre := normalized(rawGenre)
		if genre == "" || seen[genre] {
			continue
		}
		seen[genre] = true
		result = append(result, fmt.Sprintf("genre:%q", genre))

		if len(result) == maxQueries {
			return result
		}
	}

	if len(result) == 0 {
		result = append(result, "new music")
	}

	return result
}

func spotifyTrackCandidate(track spotify.Track) (entity.SpotifyCandidate, bool) {
	if strings.TrimSpace(track.ID) == "" || strings.TrimSpace(track.Name) == "" || len(track.Artists) == 0 {
		return entity.SpotifyCandidate{}, false
	}
	artist := track.Artists[0]
	if strings.TrimSpace(artist.ID) == "" || strings.TrimSpace(artist.Name) == "" {
		return entity.SpotifyCandidate{}, false
	}

	imageURL := ""
	if len(track.Album.Images) > 0 {
		imageURL = track.Album.Images[0].URL
	}
	releaseDate := parseSpotifyReleaseDate(track.Album.ReleaseDate)
	return entity.SpotifyCandidate{
		Artist: entity.Artist{
			SpotifyArtistID: artist.ID,
			Name:            artist.Name,
			SpotifyURL:      "https://open.spotify.com/artist/" + artist.ID,
		},
		Song: entity.Song{
			SpotifyTrackID: track.ID,
			Name:           track.Name,
			AlbumName:      track.Album.Name,
			ImageURL:       imageURL,
			SpotifyURL:     track.ExternalURLs["spotify"],
			PreviewURL:     track.PreviewURL,
			DurationMs:     track.DurationMS,
			Popularity:     track.Popularity,
			ReleaseDate:    releaseDate,
			Explicit:       track.Explicit,
		},
	}, true
}

func parseSpotifyReleaseDate(value string) time.Time {
	for _, layout := range []string{"2006-01-02", "2006-01", "2006"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Unix(0, 0).UTC()
}
