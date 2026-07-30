package usecase

import (
	"reflect"
	"testing"
	"time"
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
