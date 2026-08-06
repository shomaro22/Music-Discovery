package usecase

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"music-discovery/backend/entity"
	"music-discovery/backend/repository"
	"music-discovery/backend/spotify"
)

type CuratedCatalogImportResult struct {
	Defined  int
	Existing int
	Imported int
	NotFound int
}

type ICuratedCatalogImportUsecase interface {
	Import(context.Context, []CuratedCatalogSeed) (CuratedCatalogImportResult, error)
}

type curatedCatalogImportUsecase struct {
	client     spotify.IClient
	repository repository.ICuratedCatalogRepository
}

func NewCuratedCatalogImportUsecase(
	client spotify.IClient,
	repository repository.ICuratedCatalogRepository,
) ICuratedCatalogImportUsecase {
	return &curatedCatalogImportUsecase{client: client, repository: repository}
}

func (u *curatedCatalogImportUsecase) Import(
	ctx context.Context,
	seeds []CuratedCatalogSeed,
) (CuratedCatalogImportResult, error) {
	result := CuratedCatalogImportResult{Defined: len(seeds)}
	if len(seeds) == 0 {
		return result, nil
	}

	existing, err := u.repository.FindApprovedSongKeys(ctx)
	if err != nil {
		return CuratedCatalogImportResult{}, entity.ErrCuratedCatalogImportFailed
	}

	values := make([]entity.SpotifyCandidate, 0, len(seeds))
	seen := make(map[string]struct{}, len(seeds))
	for _, seed := range seeds {
		key := curatedCatalogKey(seed.Artist, seed.Track)
		if key == "|" {
			continue
		}
		if _, duplicated := seen[key]; duplicated {
			continue
		}
		seen[key] = struct{}{}
		if existing[key] {
			result.Existing++
			continue
		}

		candidate, found, searchErr := u.resolveSeed(ctx, seed)
		if searchErr != nil {
			return CuratedCatalogImportResult{}, entity.ErrCuratedCatalogImportFailed
		}
		if !found {
			result.NotFound++
			continue
		}
		candidate.Song.ReviewStatus = entity.ContentReviewStatusApproved
		candidate.TagSlugs = append([]string(nil), seed.TagSlugs...)
		values = append(values, candidate)
	}

	if len(values) == 0 {
		return result, nil
	}
	imported, err := u.repository.UpsertApprovedCandidates(ctx, values)
	if err != nil {
		return CuratedCatalogImportResult{}, entity.ErrCuratedCatalogImportFailed
	}
	result.Imported = imported
	return result, nil
}

func (u *curatedCatalogImportUsecase) resolveSeed(
	ctx context.Context,
	seed CuratedCatalogSeed,
) (entity.SpotifyCandidate, bool, error) {
	query := fmt.Sprintf(`track:%q artist:%q`, strings.TrimSpace(seed.Track), strings.TrimSpace(seed.Artist))
	result, err := u.client.SearchTracks(ctx, query, 10)
	if err != nil {
		if errors.Is(err, spotify.ErrUnavailable) {
			return entity.SpotifyCandidate{}, false, err
		}
		return entity.SpotifyCandidate{}, false, err
	}

	matches := make([]spotify.Track, 0, len(result.Items))
	for _, track := range result.Items {
		if curatedTrackMatches(track, seed) {
			matches = append(matches, track)
		}
	}
	if len(matches) == 0 {
		return entity.SpotifyCandidate{}, false, nil
	}
	sort.SliceStable(matches, func(i, j int) bool {
		return matches[i].Popularity > matches[j].Popularity
	})
	candidate, ok := spotifyTrackCandidate(matches[0])
	return candidate, ok, nil
}

func curatedTrackMatches(track spotify.Track, seed CuratedCatalogSeed) bool {
	if curatedTrackTitle(track.Name) != curatedTrackTitle(seed.Track) {
		return false
	}
	wantedArtist := curatedCatalogText(seed.Artist)
	for _, artist := range track.Artists {
		if curatedCatalogText(artist.Name) == wantedArtist {
			return true
		}
	}
	return false
}

func curatedTrackTitle(value string) string {
	value = curatedCatalogText(value)
	if strings.Contains(value, "remaster") {
		if index := strings.Index(value, " - "); index >= 0 {
			return strings.TrimSpace(value[:index])
		}
		if index := strings.Index(value, " (remaster"); index >= 0 {
			return strings.TrimSpace(value[:index])
		}
	}
	return value
}

func curatedCatalogText(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "　", " ")
	value = strings.ReplaceAll(value, "’", "'")
	return strings.Join(strings.Fields(value), " ")
}

func curatedCatalogKey(artist, track string) string {
	return curatedCatalogText(artist) + "|" + curatedTrackTitle(track)
}
