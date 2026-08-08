package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"music-discovery/backend/entity"
)

func TestRecommendationRepositoryFindCandidatesFiltersReviewStatus(t *testing.T) {
	db := openTestPostgres(t)
	if err := db.AutoMigrate(&entity.Artist{}, &entity.Song{}, &entity.Tag{}, &entity.SongTag{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("TRUNCATE TABLE song_tags, tags, songs, artists CASCADE").Error; err != nil {
		t.Fatal(err)
	}

	artist := entity.Artist{ID: uuid.New(), SpotifyArtistID: uuid.NewString(), Name: "Artist", SpotifyURL: "https://open.spotify.com/artist/test"}
	if err := db.Create(&artist).Error; err != nil {
		t.Fatal(err)
	}
	genre := entity.Tag{ID: uuid.New(), Name: "Rock", Slug: "rock", Category: "genre"}
	if err := db.Create(&genre).Error; err != nil {
		t.Fatal(err)
	}

	statuses := []entity.ContentReviewStatus{
		entity.ContentReviewStatusPending,
		entity.ContentReviewStatusRejected,
		entity.ContentReviewStatusApproved,
		entity.ContentReviewStatusAutoApproved,
	}
	ids := make(map[entity.ContentReviewStatus]uuid.UUID, len(statuses))
	for _, status := range statuses {
		song := entity.Song{
			ID: uuid.New(), ArtistID: artist.ID, SpotifyTrackID: uuid.NewString(),
			Name: string(status), AlbumName: "Album", SpotifyURL: "https://open.spotify.com/track/" + uuid.NewString(),
			ReviewStatus: status,
		}
		if err := db.Create(&song).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&entity.SongTag{ID: uuid.New(), SongID: song.ID, TagID: genre.ID, Weight: 1}).Error; err != nil {
			t.Fatal(err)
		}
		ids[status] = song.ID
	}

	candidates, err := NewRecommendationRepository(db).FindCandidates(context.Background(), RecommendationCandidateCondition{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	got := make(map[uuid.UUID]bool, len(candidates))
	for _, candidate := range candidates {
		got[candidate.Song.ID] = true
	}
	if got[ids[entity.ContentReviewStatusPending]] {
		t.Fatal("PENDING song must not be a recommendation candidate")
	}
	if got[ids[entity.ContentReviewStatusRejected]] {
		t.Fatal("REJECTED song must not be a recommendation candidate")
	}
	if !got[ids[entity.ContentReviewStatusApproved]] {
		t.Fatal("APPROVED song must be a recommendation candidate")
	}
	if !got[ids[entity.ContentReviewStatusAutoApproved]] {
		t.Fatal("AUTO_APPROVED song must be a recommendation candidate")
	}
}
