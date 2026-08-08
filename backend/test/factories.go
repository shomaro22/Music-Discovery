package testutil

import (
	"fmt"
	"sync/atomic"
	"time"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
)

var sequence atomic.Uint64

func next(prefix string) string { return fmt.Sprintf("%s-%d", prefix, sequence.Add(1)) }

func User(overrides ...func(*entity.User)) entity.User {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	value := entity.User{ID: uuid.New(), Name: "Test User", Email: next("user") + "@example.test", PasswordHash: "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", Role: entity.UserRoleUser, Status: entity.UserStatusActive, CreatedAt: now, UpdatedAt: now}
	for _, override := range overrides {
		override(&value)
	}
	return value
}

func SuspendedUser() entity.User {
	return User(func(value *entity.User) {
		value.Status = entity.UserStatusSuspended
		value.SuspendedReason = "test suspension"
	})
}

func ContentAdmin() entity.User {
	return User(func(value *entity.User) { value.Role = entity.UserRoleContentAdmin })
}
func SystemAdmin() entity.User {
	return User(func(value *entity.User) { value.Role = entity.UserRoleSystemAdmin })
}

func Artist(overrides ...func(*entity.Artist)) entity.Artist {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	value := entity.Artist{ID: uuid.New(), SpotifyArtistID: next("spotify-artist"), Name: next("Artist"), SpotifyURL: "https://open.spotify.com/artist/test", Popularity: 50, CreatedAt: now, UpdatedAt: now}
	for _, override := range overrides {
		override(&value)
	}
	return value
}

func Song(artist entity.Artist, overrides ...func(*entity.Song)) entity.Song {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	value := entity.Song{ID: uuid.New(), ArtistID: artist.ID, Artist: artist, SpotifyTrackID: next("spotify-track"), Name: next("Song"), AlbumName: "Test Album", SpotifyURL: "https://open.spotify.com/track/test", DurationMs: 180000, Popularity: 50, ReleaseDate: now, ReviewStatus: entity.ContentReviewStatusApproved, CreatedAt: now, UpdatedAt: now}
	for _, override := range overrides {
		override(&value)
	}
	return value
}

func InvalidSong(artist entity.Artist) entity.Song {
	return Song(artist, func(value *entity.Song) { value.ReviewStatus = entity.ContentReviewStatusRejected })
}
func SongWithoutSpotifyURL(artist entity.Artist) entity.Song {
	return Song(artist, func(value *entity.Song) { value.SpotifyURL = "" })
}

func Tag(category string) entity.Tag {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	slug := next("tag")
	return entity.Tag{ID: uuid.New(), Name: slug, Slug: slug, Category: category, CreatedAt: now, UpdatedAt: now}
}

// Genre returns a valid genre tag because genres are represented by Tag entities in the current model.
func Genre() entity.Tag { return Tag("genre") }

func Question(code string, sortOrder int) entity.Question {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	max := 1
	return entity.Question{ID: uuid.New(), Code: code, Title: "Test question", AnswerType: entity.QuestionAnswerTypeSingle, MinSelections: 1, MaxSelections: &max, SortOrder: sortOrder, IsActive: true, CreatedAt: now, UpdatedAt: now}
}

func QuestionChoice(question entity.Question, code string, sortOrder int) entity.QuestionChoice {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return entity.QuestionChoice{ID: uuid.New(), QuestionID: question.ID, Code: code, Label: code, SortOrder: sortOrder, IsActive: true, CreatedAt: now, UpdatedAt: now}
}

func UserProfile(user entity.User) entity.UserProfile {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return entity.UserProfile{ID: uuid.New(), UserID: user.ID, Timezone: "Asia/Tokyo", DiscoveryLevel: "DISCOVERY_MEDIUM", CreatedAt: now, UpdatedAt: now}
}

func QuestionAnswerSession(user entity.User, answeredAt time.Time) entity.QuestionAnswerSession {
	return entity.QuestionAnswerSession{ID: uuid.New(), UserID: user.ID, AnsweredAt: answeredAt.UTC(), CreatedAt: answeredAt.UTC()}
}

func Recommendation(user entity.User, source entity.RecommendationSource, recommendedAt time.Time) entity.Recommendation {
	value := entity.Recommendation{
		ID:                  uuid.New(),
		UserID:              user.ID,
		Source:              source,
		RecommendedAt:       recommendedAt.UTC(),
		AlgorithmVersion:    "test-v1",
		WeightConfigVersion: "weights-v1",
		CreatedAt:           recommendedAt.UTC(),
	}
	if source == entity.RecommendationSourceDailyDiscovery {
		date := TargetDateForTimezone(recommendedAt, "Asia/Tokyo")
		value.TargetDate = &date
	}
	return value
}

func RecommendationDetail(recommendation entity.Recommendation, song entity.Song, recommendationType entity.RecommendationType, position int) entity.RecommendationDetail {
	now := recommendation.RecommendedAt.UTC()
	return entity.RecommendationDetail{
		ID:               uuid.New(),
		RecommendationID: recommendation.ID,
		SongID:           song.ID,
		Type:             recommendationType,
		Score:            1,
		ScoreBreakdown:   []byte(`{"test":1}`),
		Position:         position,
		Reason:           "Test recommendation reason",
		Song:             song,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func Feedback(user entity.User, detail entity.RecommendationDetail, feedbackType entity.FeedbackType) entity.Feedback {
	now := detail.CreatedAt.UTC()
	return entity.Feedback{
		ID:                     uuid.New(),
		UserID:                 user.ID,
		RecommendationDetailID: detail.ID,
		SongID:                 detail.SongID,
		ArtistID:               detail.Song.ArtistID,
		Type:                   feedbackType,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
}

func RecommendationHistory(user entity.User, source entity.RecommendationSource, recommendedAt time.Time) entity.Recommendation {
	return Recommendation(user, source, recommendedAt)
}

func RecommendationWithin30Days(user entity.User, now time.Time) entity.Recommendation {
	return Recommendation(user, entity.RecommendationSourceNormal, now.AddDate(0, 0, -29))
}

func RecommendationOlderThan30Days(user entity.User, now time.Time) entity.Recommendation {
	return Recommendation(user, entity.RecommendationSourceNormal, now.AddDate(0, 0, -31))
}

func UserAnswer(session entity.QuestionAnswerSession, question entity.Question) entity.UserAnswer {
	now := session.AnsweredAt.UTC()
	return entity.UserAnswer{
		ID:              uuid.New(),
		AnswerSessionID: session.ID,
		QuestionID:      question.ID,
		CreatedAt:       now,
	}
}

func UserAnswerChoice(answer entity.UserAnswer, choice entity.QuestionChoice) entity.UserAnswerChoice {
	return entity.UserAnswerChoice{
		ID:               uuid.New(),
		UserAnswerID:     answer.ID,
		QuestionChoiceID: choice.ID,
		CreatedAt:        answer.CreatedAt.UTC(),
	}
}

func GenreWeight(profile entity.UserProfile, choice entity.QuestionChoice, questionWeight, feedbackWeight float64) entity.UserGenreWeight {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return entity.UserGenreWeight{
		ID: uuid.New(), ProfileID: profile.ID, ChoiceID: choice.ID,
		QuestionWeight: questionWeight, FeedbackWeight: feedbackWeight,
		Weight: questionWeight + feedbackWeight, CreatedAt: now, UpdatedAt: now,
	}
}

func ElementWeight(profile entity.UserProfile, choice entity.QuestionChoice, questionWeight, feedbackWeight float64) entity.UserElementWeight {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return entity.UserElementWeight{
		ID: uuid.New(), ProfileID: profile.ID, ChoiceID: choice.ID,
		QuestionWeight: questionWeight, FeedbackWeight: feedbackWeight,
		Weight: questionWeight + feedbackWeight, CreatedAt: now, UpdatedAt: now,
	}
}

func AtmosphereWeight(profile entity.UserProfile, choice entity.QuestionChoice, questionWeight, feedbackWeight float64) entity.UserAtmosphereWeight {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return entity.UserAtmosphereWeight{
		ID: uuid.New(), ProfileID: profile.ID, ChoiceID: choice.ID,
		QuestionWeight: questionWeight, FeedbackWeight: feedbackWeight,
		Weight: questionWeight + feedbackWeight, CreatedAt: now, UpdatedAt: now,
	}
}

func ArtistWeight(user entity.User, artist entity.Artist, weight float64) entity.UserArtistWeight {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return entity.UserArtistWeight{ID: uuid.New(), UserID: user.ID, ArtistID: artist.ID, Weight: weight, CreatedAt: now, UpdatedAt: now}
}

func SongWeight(user entity.User, song entity.Song, weight float64) entity.UserSongWeight {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return entity.UserSongWeight{ID: uuid.New(), UserID: user.ID, SongID: song.ID, Weight: weight, CreatedAt: now, UpdatedAt: now}
}

func DailyDiscovery(user entity.User, at time.Time, timezone string) entity.Recommendation {
	value := Recommendation(user, entity.RecommendationSourceDailyDiscovery, at)
	date := TargetDateForTimezone(at, timezone)
	value.TargetDate = &date
	return value
}

func TargetDateForTimezone(at time.Time, timezone string) time.Time {
	location, err := time.LoadLocation(timezone)
	if err != nil || location == nil {
		if timezone == "Asia/Tokyo" {
			location = time.FixedZone("Asia/Tokyo", 9*60*60)
		} else {
			location = time.UTC
		}
	}
	local := at.In(location)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}
