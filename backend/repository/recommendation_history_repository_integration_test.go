package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"music-discovery/backend/entity"
)

func TestRecommendationHistoryRepositoryFiltersOrdersAndLoadsFeedback(t *testing.T) {
	db := openTestPostgres(t)
	if err := db.AutoMigrate(&entity.Artist{}, &entity.Song{}, &entity.Recommendation{}, &entity.RecommendationDetail{}, &entity.Feedback{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("TRUNCATE TABLE feedbacks, recommendation_details, recommendations, songs, artists CASCADE").Error; err != nil {
		t.Fatal(err)
	}

	userID := uuid.New()
	otherID := uuid.New()
	artist := entity.Artist{ID: uuid.New(), SpotifyArtistID: uuid.NewString(), Name: "Artist"}
	if err := db.Create(&artist).Error; err != nil {
		t.Fatal(err)
	}
	songs := []entity.Song{{ID: uuid.New(), SpotifyTrackID: uuid.NewString(), Name: "One", ArtistID: artist.ID}, {ID: uuid.New(), SpotifyTrackID: uuid.NewString(), Name: "Two", ArtistID: artist.ID}, {ID: uuid.New(), SpotifyTrackID: uuid.NewString(), Name: "Three", ArtistID: artist.ID}}
	if err := db.Create(&songs).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	makeRecommendation := func(owner uuid.UUID, source entity.RecommendationSource, at time.Time) entity.Recommendation {
		r := entity.Recommendation{ID: uuid.New(), UserID: owner, Source: source, RecommendedAt: at, AlgorithmVersion: "v1"}
		for i, song := range songs {
			r.Details = append(r.Details, entity.RecommendationDetail{ID: uuid.New(), SongID: song.ID, Type: []entity.RecommendationType{entity.RecommendationTypeYourMatch, entity.RecommendationTypeNewFavorite, entity.RecommendationTypeTrySomethingNew}[i], Position: i + 1, Reason: "reason", Score: 1})
		}
		return r
	}
	older := makeRecommendation(userID, entity.RecommendationSourceNormal, now.Add(-time.Hour))
	newer := makeRecommendation(userID, entity.RecommendationSourceDailyDiscovery, now)
	other := makeRecommendation(otherID, entity.RecommendationSourceNormal, now.Add(time.Hour))
	for _, value := range []*entity.Recommendation{&older, &newer, &other} {
		if err := db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&entity.Feedback{ID: uuid.New(), UserID: userID, RecommendationDetailID: newer.Details[0].ID, SongID: songs[0].ID, ArtistID: artist.ID, Type: entity.FeedbackTypeLike}).Error; err != nil {
		t.Fatal(err)
	}

	values, total, err := NewRecommendationRepository(db).ListHistory(context.Background(), RecommendationHistoryCondition{UserID: userID, Page: 1, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(values) != 2 || values[0].ID != newer.ID {
		t.Fatalf("total=%d values=%+v", total, values)
	}
	if values[0].Details[0].FeedbackStatus == nil || *values[0].Details[0].FeedbackStatus != entity.FeedbackStatusLike {
		t.Fatalf("feedback=%v", values[0].Details[0].FeedbackStatus)
	}
}
