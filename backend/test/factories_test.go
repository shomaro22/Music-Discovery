package testutil

import (
	"testing"
	"time"

	"music-discovery/backend/entity"

	"golang.org/x/crypto/bcrypt"
)

func TestFactoriesCreateIndependentValidDefaults(t *testing.T) {
	first := User()
	second := User()
	if first.ID == second.ID || first.Email == second.Email {
		t.Fatal("user factory must create unique values")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(first.PasswordHash), []byte("password")); err != nil {
		t.Fatalf("user factory password hash is not a valid bcrypt hash: %v", err)
	}
	artist := Artist()
	song := Song(artist)
	if song.ArtistID != artist.ID || song.ReviewStatus != entity.ContentReviewStatusApproved {
		t.Fatalf("unexpected song factory result: %+v", song)
	}
	if SuspendedUser().Status != entity.UserStatusSuspended {
		t.Fatal("suspended user factory must set suspended status")
	}
	if SongWithoutSpotifyURL(artist).SpotifyURL != "" {
		t.Fatal("Spotify URL override was not applied")
	}
	if genre := Genre(); genre.Category != "genre" {
		t.Fatalf("genre factory category=%q want=genre", genre.Category)
	}
}

func TestTargetDateForTimezoneUsesCalendarDate(t *testing.T) {
	instant := time.Date(2026, 8, 7, 16, 30, 0, 0, time.UTC) // 2026-08-08 01:30 JST
	date := TargetDateForTimezone(instant, "Asia/Tokyo")
	if got := date.Format("2006-01-02"); got != "2026-08-08" {
		t.Fatalf("target date=%s want=2026-08-08", got)
	}
}
