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

type SpotifyCandidateImportResult struct {
	Processed    int
	Created      int
	AutoApproved int
	Pending      int
	Skipped      int
}

type ISpotifyCandidateImportUsecase interface {
	Import(context.Context, []string) (SpotifyCandidateImportResult, error)
}

type spotifyCandidateImportUsecase struct {
	client     spotify.IClient
	repository repository.ISpotifyCandidateRepository
	config     SpotifyCandidateImportConfig
}

func NewSpotifyCandidateImportUsecase(client spotify.IClient, repository repository.ISpotifyCandidateRepository, config SpotifyCandidateImportConfig) ISpotifyCandidateImportUsecase {
	return &spotifyCandidateImportUsecase{client: client, repository: repository, config: config}
}

func (u *spotifyCandidateImportUsecase) Import(ctx context.Context, genres []string) (SpotifyCandidateImportResult, error) {
	queries := spotifyCandidateQueries(genres, u.config.MaxQueries)
	limit := u.config.SearchLimit
	if limit < 1 || limit > 10 {
		limit = 10
	}

	// Search farther than the first page when the popular results already exist
	// in the database. Five pages means at most 50 tracks per query.
	const maxPagesPerQuery = 5

	byTrackID := make(map[string]entity.SpotifyCandidate)
	processed := 0
	for _, query := range queries {
		tagSlugs := spotifyCandidateTagSlugs(query)
		for page := 0; page < maxPagesPerQuery; page++ {
			offset := page * limit
			result, err := u.searchTracks(ctx, query, limit, offset)
			if err != nil {
				return SpotifyCandidateImportResult{}, entity.ErrSpotifyCandidateImportFailed
			}
			processed += len(result.Items)
			if len(result.Items) == 0 {
				break
			}

			trackIDs := make([]string, 0, len(result.Items))
			for _, track := range result.Items {
				if strings.TrimSpace(track.ID) != "" {
					trackIDs = append(trackIDs, track.ID)
				}
			}
			existingIDs, err := u.repository.FindExistingSpotifyTrackIDs(ctx, trackIDs)
			if err != nil {
				return SpotifyCandidateImportResult{}, entity.ErrSpotifyCandidateImportFailed
			}

			newOnPage := 0
			for _, track := range result.Items {
				_, alreadyExists := existingIDs[track.ID]
				candidate, ok := spotifyTrackCandidate(track)
				if !ok {
					continue
				}
				candidate.TagSlugs = append([]string(nil), tagSlugs...)
				if existing, found := byTrackID[track.ID]; found {
					candidate.TagSlugs = mergeStrings(existing.TagSlugs, candidate.TagSlugs)
				} else if !alreadyExists {
					newOnPage++
				}
				byTrackID[track.ID] = candidate
			}

			// Stop when Spotify has no more pages. Continue when this page only
			// contained duplicates so the next offset can provide fresh tracks.
			if len(result.Items) < limit {
				break
			}
			if newOnPage > 0 && len(byTrackID) >= limit*len(queries) {
				break
			}
		}
	}

	values := make([]entity.SpotifyCandidate, 0, len(byTrackID))
	for _, value := range byTrackID {
		value.Song.ReviewStatus = entity.ContentReviewStatusPending
		if isHighConfidenceSpotifyCandidate(value) {
			value.Song.ReviewStatus = entity.ContentReviewStatusAutoApproved
		}
		values = append(values, value)
	}

	saved, err := u.repository.UpsertCandidates(ctx, values)
	if err != nil {
		return SpotifyCandidateImportResult{}, entity.ErrSpotifyCandidateImportFailed
	}
	return SpotifyCandidateImportResult{
		Processed:    processed,
		Created:      saved.Created,
		AutoApproved: saved.AutoApproved,
		Pending:      saved.Pending,
		Skipped:      saved.Skipped,
	}, nil
}

func (u *spotifyCandidateImportUsecase) searchTracks(ctx context.Context, query string, limit, offset int) (spotify.SearchTracksResult, error) {
	if client, ok := u.client.(spotify.ITrackSearchWithOffset); ok {
		return client.SearchTracksWithOffset(ctx, query, limit, offset)
	}
	if offset > 0 {
		return spotify.SearchTracksResult{}, nil
	}
	return u.client.SearchTracks(ctx, query, limit)
}

func spotifyCandidateQueries(genres []string, maxQueries int) []string {
	if maxQueries <= 0 {
		maxQueries = 3
	}
	result := make([]string, 0, maxQueries)
	seen := map[string]bool{}
	for _, rawGenre := range genres {
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

func spotifyCandidateGenreSlug(query string) string {
	const prefix = `genre:"`
	if !strings.HasPrefix(query, prefix) || !strings.HasSuffix(query, `"`) {
		return ""
	}
	genre := strings.TrimSuffix(strings.TrimPrefix(query, prefix), `"`)
	genre = normalized(genre)
	if genre == "" {
		return ""
	}
	return "genre-" + genre
}

// spotifyCandidateTagSlugs maps only the broad characteristics that can be
// conservatively inferred from the Spotify genre query. This preserves the
// Try Something New rule requiring a Mood/Element/Atmosphere contact while
// avoiding a fallback that ignores that rule.
func spotifyCandidateTagSlugs(query string) []string {
	genreSlug := spotifyCandidateGenreSlug(query)
	if genreSlug == "" {
		return nil
	}

	mapping := map[string][]string{
		"genre-pop":         {"mood-uplifting", "element-vocal", "atmosphere-bright"},
		"genre-rock":        {"mood-uplifting", "element-rhythm", "atmosphere-refreshing"},
		"genre-alternative": {"mood-extraordinary", "element-worldview", "atmosphere-deep"},
		"genre-metal":       {"mood-extraordinary", "element-rhythm", "atmosphere-deep"},
		"genre-hip-hop":     {"mood-uplifting", "element-lyrics", "element-rhythm", "atmosphere-night"},
		"genre-jazz":        {"mood-calm", "element-melody", "atmosphere-night"},
		"genre-electronic":  {"mood-focus", "element-rhythm", "atmosphere-night"},
		"genre-classical":   {"mood-calm", "element-melody", "atmosphere-cinematic"},
		"genre-anime":       {"mood-extraordinary", "element-worldview", "atmosphere-cinematic"},
		"genre-k-pop":       {"mood-uplifting", "element-vocal", "element-rhythm", "atmosphere-bright"},
	}

	return append([]string{genreSlug}, mapping[genreSlug]...)
}

func mergeStrings(left, right []string) []string {
	result := make([]string, 0, len(left)+len(right))
	seen := make(map[string]struct{}, len(left)+len(right))
	for _, values := range [][]string{left, right} {
		for _, value := range values {
			if value == "" {
				continue
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	return result
}

func isHighConfidenceSpotifyCandidate(value entity.SpotifyCandidate) bool {
	return strings.TrimSpace(value.Artist.SpotifyArtistID) != "" &&
		strings.TrimSpace(value.Artist.Name) != "" &&
		strings.TrimSpace(value.Artist.SpotifyURL) != "" &&
		strings.TrimSpace(value.Song.SpotifyTrackID) != "" &&
		strings.TrimSpace(value.Song.Name) != "" &&
		strings.TrimSpace(value.Song.AlbumName) != "" &&
		strings.TrimSpace(value.Song.ImageURL) != "" &&
		strings.TrimSpace(value.Song.SpotifyURL) != "" &&
		value.Song.DurationMs > 0 &&
		hasSpotifyCandidateContactTags(value.TagSlugs)
}

func hasSpotifyCandidateContactTags(slugs []string) bool {
	hasGenre := false
	hasContact := false
	for _, slug := range slugs {
		switch {
		case strings.HasPrefix(slug, "genre-"):
			hasGenre = true
		case strings.HasPrefix(slug, "mood-"), strings.HasPrefix(slug, "element-"), strings.HasPrefix(slug, "atmosphere-"):
			hasContact = true
		}
	}
	return hasGenre && hasContact
}
