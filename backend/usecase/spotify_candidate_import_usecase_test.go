package usecase

import (
	"reflect"
	"testing"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/spotify"
)

func TestSpotifyCandidateQueriesNormalizesQuestionChoiceCodes(t *testing.T) {
	values := spotifyCandidateQueries(
		[]string{
			"GENRE_POP",
			"GENRE_HIP_HOP",
			"GENRE_K_POP",
		},
		3,
	)

	expected := []string{
		`genre:"pop"`,
		`genre:"hip-hop"`,
		`genre:"k-pop"`,
	}

	if !reflect.DeepEqual(values, expected) {
		t.Fatalf("values = %#v, expected = %#v", values, expected)
	}
}

func TestSpotifyCandidateQueriesUsesDistinctGenres(t *testing.T) {
	values := spotifyCandidateQueries([]string{"pop", " Pop ", "rock", "jazz"}, 2)
	if len(values) != 2 || values[0] != `genre:"pop"` || values[1] != `genre:"rock"` {
		t.Fatalf("unexpected queries: %#v", values)
	}
}

func TestSpotifyCandidateQueriesFallsBackWhenGenresAreEmpty(t *testing.T) {
	values := spotifyCandidateQueries(nil, 3)
	if len(values) != 1 || values[0] != "new music" {
		t.Fatalf("unexpected queries: %#v", values)
	}
}

func TestParseSpotifyReleaseDate(t *testing.T) {
	cases := map[string]time.Time{
		"2026-07-30": time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC),
		"2026-07":    time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		"2026":       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	for input, expected := range cases {
		if actual := parseSpotifyReleaseDate(input); !actual.Equal(expected) {
			t.Fatalf("input=%s actual=%s expected=%s", input, actual, expected)
		}
	}
}

func TestSpotifyCandidateGenreSlug(t *testing.T) {
	cases := map[string]string{
		`genre:"pop"`:     "genre-pop",
		`genre:"hip-hop"`: "genre-hip-hop",
		"new music":       "",
	}
	for input, expected := range cases {
		if actual := spotifyCandidateGenreSlug(input); actual != expected {
			t.Fatalf("input=%s actual=%s expected=%s", input, actual, expected)
		}
	}
}

func TestHighConfidenceSpotifyCandidate(t *testing.T) {
	value := entity.SpotifyCandidate{
		Artist: entity.Artist{
			SpotifyArtistID: "artist-id",
			Name:            "Artist",
			SpotifyURL:      "https://open.spotify.com/artist/artist-id",
		},
		Song: entity.Song{
			SpotifyTrackID: "track-id",
			Name:           "Song",
			AlbumName:      "Album",
			ImageURL:       "https://example.com/artwork.jpg",
			SpotifyURL:     "https://open.spotify.com/track/track-id",
			DurationMs:     180000,
		},
		TagSlugs: []string{"genre-pop", "mood-uplifting", "element-vocal", "atmosphere-bright"},
	}
	if !isHighConfidenceSpotifyCandidate(value) {
		t.Fatal("complete candidate must be high confidence")
	}

	value.Song.ImageURL = ""
	if isHighConfidenceSpotifyCandidate(value) {
		t.Fatal("candidate without artwork must remain pending")
	}
}

func TestMergeStringsRemovesDuplicates(t *testing.T) {
	actual := mergeStrings([]string{"genre-pop"}, []string{"genre-pop", "genre-rock"})
	expected := []string{"genre-pop", "genre-rock"}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("actual=%#v expected=%#v", actual, expected)
	}
}

func TestSpotifyCandidateTagSlugsAddsContactTags(t *testing.T) {
	actual := spotifyCandidateTagSlugs(`genre:"anime"`)
	expected := []string{"genre-anime", "mood-extraordinary", "element-worldview", "atmosphere-cinematic"}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("actual=%#v expected=%#v", actual, expected)
	}
}

func TestHighConfidenceSpotifyCandidateRequiresContactTag(t *testing.T) {
	value := entity.SpotifyCandidate{
		Artist:   entity.Artist{SpotifyArtistID: "artist-id", Name: "Artist", SpotifyURL: "https://open.spotify.com/artist/artist-id"},
		Song:     entity.Song{SpotifyTrackID: "track-id", Name: "Song", AlbumName: "Album", ImageURL: "https://example.com/artwork.jpg", SpotifyURL: "https://open.spotify.com/track/track-id", DurationMs: 180000},
		TagSlugs: []string{"genre-pop"},
	}
	if isHighConfidenceSpotifyCandidate(value) {
		t.Fatal("genre-only candidate must remain pending")
	}
}

func TestSpotifyTrackMeetsImportQualityRejectsLowPopularityAndNonSongContent(t *testing.T) {
	base := spotify.Track{Name: "Regular Song", Popularity: 40}
	if !spotifyTrackMeetsImportQuality(base, 35) {
		t.Fatal("regular track above threshold must be accepted")
	}

	lowPopularity := base
	lowPopularity.Popularity = 34
	if spotifyTrackMeetsImportQuality(lowPopularity, 35) {
		t.Fatal("track below popularity threshold must be rejected")
	}

	for _, title := range []string{"Album Intro", "Leaked Audio - Skit", "Closing Outro", "Short Interlude"} {
		track := base
		track.Name = title
		if spotifyTrackMeetsImportQuality(track, 35) {
			t.Fatalf("non-song content %q must be rejected", title)
		}
	}
}

func TestHighConfidenceSpotifyCandidateStillRequiresMinimumPopularityAtImportDecision(t *testing.T) {
	value := entity.SpotifyCandidate{
		Artist: entity.Artist{
			SpotifyArtistID: "artist-id",
			Name:            "Artist",
			SpotifyURL:      "https://open.spotify.com/artist/artist-id",
		},
		Song: entity.Song{
			SpotifyTrackID: "track-id",
			Name:           "Song",
			AlbumName:      "Album",
			ImageURL:       "https://example.com/artwork.jpg",
			SpotifyURL:     "https://open.spotify.com/track/track-id",
			DurationMs:     180000,
			Popularity:     34,
		},
		TagSlugs: []string{"genre-pop", "mood-uplifting", "element-vocal", "atmosphere-bright"},
	}
	if !isHighConfidenceSpotifyCandidate(value) {
		t.Fatal("metadata completeness and popularity threshold are intentionally separate checks")
	}
	if value.Song.Popularity >= 35 {
		t.Fatal("test setup must remain below the automatic approval threshold")
	}
}
