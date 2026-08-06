package usecase

import (
	"context"
	"reflect"
	"testing"

	"music-discovery/backend/entity"
	"music-discovery/backend/spotify"
)

type curatedCatalogClientStub struct {
	queries []string
	results map[string]spotify.SearchTracksResult
	err     error
}

func (*curatedCatalogClientStub) SearchArtists(context.Context, string, int) (spotify.SearchArtistsResult, error) {
	return spotify.SearchArtistsResult{}, nil
}

func (s *curatedCatalogClientStub) SearchTracks(_ context.Context, query string, _ int) (spotify.SearchTracksResult, error) {
	s.queries = append(s.queries, query)
	if s.err != nil {
		return spotify.SearchTracksResult{}, s.err
	}
	return s.results[query], nil
}

type curatedCatalogRepositoryStub struct {
	existing map[string]bool
	saved    []entity.SpotifyCandidate
	err      error
}

func (s *curatedCatalogRepositoryStub) FindApprovedSongKeys(context.Context) (map[string]bool, error) {
	return s.existing, s.err
}

func (s *curatedCatalogRepositoryStub) UpsertApprovedCandidates(_ context.Context, values []entity.SpotifyCandidate) (int, error) {
	if s.err != nil {
		return 0, s.err
	}
	s.saved = append(s.saved, values...)
	return len(values), nil
}

func TestDefaultCuratedCatalogSeedsHasBalancedFoundation(t *testing.T) {
	seeds := DefaultCuratedCatalogSeeds()
	if len(seeds) != 120 {
		t.Fatalf("seed count=%d, want 120", len(seeds))
	}

	wanted := map[string]bool{
		"subgenre-j-pop":         false,
		"subgenre-melodic-punk":  false,
		"subgenre-punk-rock":     false,
		"subgenre-post-hardcore": false,
		"subgenre-metalcore":     false,
		"subgenre-heavy-metal":   false,
		"subgenre-k-pop":         false,
	}
	for _, seed := range seeds {
		for _, tag := range seed.TagSlugs {
			if _, ok := wanted[tag]; ok {
				wanted[tag] = true
			}
		}
	}
	for tag, found := range wanted {
		if !found {
			t.Fatalf("catalog does not cover %s", tag)
		}
	}
}

func TestCuratedCatalogImportSkipsExistingAndApprovesExactSpotifyMatch(t *testing.T) {
	seedExisting := CuratedCatalogSeed{Artist: "Existing Artist", Track: "Existing Song", TagSlugs: []string{"genre-pop", "mood-uplifting"}}
	seedNew := CuratedCatalogSeed{Artist: "ELLEGARDEN", Track: "Missing", TagSlugs: []string{"genre-rock", "subgenre-melodic-punk", "mood-uplifting", "element-rhythm", "atmosphere-refreshing"}}
	query := `track:"Missing" artist:"ELLEGARDEN"`
	client := &curatedCatalogClientStub{results: map[string]spotify.SearchTracksResult{
		query: {
			Items: []spotify.Track{
				{ID: "wrong", Name: "Missing", Artists: []spotify.TrackArtist{{ID: "wrong-artist", Name: "Another Artist"}}},
				{
					ID: "track-id", Name: "Missing", Popularity: 70,
					Artists:      []spotify.TrackArtist{{ID: "artist-id", Name: "ELLEGARDEN"}},
					Album:        spotify.Album{Name: "Album", ReleaseDate: "2004-01-01", Images: []spotify.Image{{URL: "https://example.com/image.jpg"}}},
					DurationMS:   180000,
					ExternalURLs: map[string]string{"spotify": "https://open.spotify.com/track/track-id"},
				},
			},
		},
	}}
	repository := &curatedCatalogRepositoryStub{existing: map[string]bool{
		curatedCatalogKey(seedExisting.Artist, seedExisting.Track): true,
	}}

	result, err := NewCuratedCatalogImportUsecase(client, repository).Import(
		context.Background(),
		[]CuratedCatalogSeed{seedExisting, seedNew},
	)
	if err != nil {
		t.Fatalf("Import error: %v", err)
	}
	if result.Defined != 2 || result.Existing != 1 || result.Imported != 1 || result.NotFound != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if !reflect.DeepEqual(client.queries, []string{query}) {
		t.Fatalf("queries=%#v", client.queries)
	}
	if len(repository.saved) != 1 {
		t.Fatalf("saved=%d, want 1", len(repository.saved))
	}
	value := repository.saved[0]
	if value.Song.ReviewStatus != entity.ContentReviewStatusApproved {
		t.Fatalf("review status=%s", value.Song.ReviewStatus)
	}
	if !reflect.DeepEqual(value.TagSlugs, seedNew.TagSlugs) {
		t.Fatalf("tags=%#v want=%#v", value.TagSlugs, seedNew.TagSlugs)
	}
}

func TestCuratedTrackMatchesAllowsRemasteredSuffixButRejectsWrongArtist(t *testing.T) {
	seed := CuratedCatalogSeed{Artist: "Green Day", Track: "Basket Case"}
	track := spotify.Track{
		Name:    "Basket Case - 4-track demo - Remastered",
		Artists: []spotify.TrackArtist{{Name: "Green Day"}},
	}
	if !curatedTrackMatches(track, seed) {
		t.Fatal("remastered suffix should preserve an exact catalog match")
	}
	track.Artists = []spotify.TrackArtist{{Name: "Cover Artist"}}
	if curatedTrackMatches(track, seed) {
		t.Fatal("wrong artist must not match")
	}
}
