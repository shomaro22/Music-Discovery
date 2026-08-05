package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"music-discovery/backend/entity"
)

type SpotifyCandidateSaveResult struct {
	Created      int
	AutoApproved int
	Pending      int
	Skipped      int
}

type ISpotifyCandidateRepository interface {
	FindExistingSpotifyTrackIDs(context.Context, []string) (map[string]struct{}, error)
	UpsertCandidates(context.Context, []entity.SpotifyCandidate) (SpotifyCandidateSaveResult, error)
}

type spotifyCandidateRepository struct{ db *gorm.DB }

func NewSpotifyCandidateRepository(db *gorm.DB) ISpotifyCandidateRepository {
	return &spotifyCandidateRepository{db: db}
}

func (r *spotifyCandidateRepository) FindExistingSpotifyTrackIDs(ctx context.Context, trackIDs []string) (map[string]struct{}, error) {
	result := make(map[string]struct{})
	if len(trackIDs) == 0 {
		return result, nil
	}

	var values []string
	if err := r.db.WithContext(ctx).
		Unscoped().
		Model(&entity.Song{}).
		Where("spotify_track_id IN ?", trackIDs).
		Pluck("spotify_track_id", &values).Error; err != nil {
		return nil, entity.ErrInternal
	}
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result, nil
}

func (r *spotifyCandidateRepository) UpsertCandidates(ctx context.Context, values []entity.SpotifyCandidate) (SpotifyCandidateSaveResult, error) {
	result := SpotifyCandidateSaveResult{}
	if len(values) == 0 {
		return result, nil
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, value := range values {
			created, status, err := insertSpotifyCandidate(tx, value)
			if err != nil {
				return err
			}
			if !created {
				result.Skipped++
				continue
			}
			result.Created++
			switch status {
			case entity.ContentReviewStatusAutoApproved:
				result.AutoApproved++
			case entity.ContentReviewStatusPending:
				result.Pending++
			}
		}
		return nil
	})
	if err != nil {
		return SpotifyCandidateSaveResult{}, err
	}
	return result, nil
}

func insertSpotifyCandidate(tx *gorm.DB, value entity.SpotifyCandidate) (bool, entity.ContentReviewStatus, error) {
	var existingSongs []entity.Song
	if err := tx.Unscoped().
		Where("spotify_track_id = ?", value.Song.SpotifyTrackID).
		Limit(1).
		Find(&existingSongs).Error; err != nil {
		return false, "", entity.ErrInternal
	}
	if len(existingSongs) > 0 {
		existing := existingSongs[0]
		// Do not rewrite managed Song metadata or updated_at. For an existing
		// AUTO_APPROVED Song, however, enrich only its inferred system Tags so
		// candidates imported before contact-tag mapping was introduced become
		// usable without being recreated.
		if existing.ReviewStatus == entity.ContentReviewStatusAutoApproved && !existing.DeletedAt.Valid {
			tags, err := findCandidateTags(tx, value.TagSlugs)
			if err != nil {
				return false, "", entity.ErrInternal
			}
			if hasRequiredCandidateTags(tags) {
				if err := upsertAutoApprovedSongTags(tx, existing.ID, tags); err != nil {
					return false, "", entity.ErrInternal
				}
				if err := upsertAutoApprovedArtistTags(tx, existing.ArtistID, tags); err != nil {
					return false, "", entity.ErrInternal
				}
			}
		}
		return false, "", nil
	}

	artist, created, err := findOrCreateSpotifyArtist(tx, value.Artist)
	if err != nil {
		return false, "", entity.ErrInternal
	}
	if !created && artist.DeletedAt.Valid {
		// A manually disabled Artist must never be restored by automatic import.
		return false, "", nil
	}

	tags, err := findCandidateTags(tx, value.TagSlugs)
	if err != nil {
		return false, "", entity.ErrInternal
	}

	song := value.Song
	song.ArtistID = artist.ID
	if song.ReviewStatus == entity.ContentReviewStatusAutoApproved && !hasRequiredCandidateTags(tags) {
		song.ReviewStatus = entity.ContentReviewStatusPending
	}

	if err := tx.Create(&song).Error; err != nil {
		return false, "", entity.ErrInternal
	}
	if song.ReviewStatus == entity.ContentReviewStatusAutoApproved {
		if err := upsertAutoApprovedSongTags(tx, song.ID, tags); err != nil {
			return false, "", entity.ErrInternal
		}
		if err := upsertAutoApprovedArtistTags(tx, artist.ID, tags); err != nil {
			return false, "", entity.ErrInternal
		}
	}
	return true, song.ReviewStatus, nil
}

func findOrCreateSpotifyArtist(tx *gorm.DB, candidate entity.Artist) (entity.Artist, bool, error) {
	var artists []entity.Artist
	if err := tx.Unscoped().
		Where("spotify_artist_id = ?", candidate.SpotifyArtistID).
		Limit(1).
		Find(&artists).Error; err != nil {
		return entity.Artist{}, false, err
	}
	if len(artists) > 0 {
		return artists[0], false, nil
	}

	if err := tx.Create(&candidate).Error; err != nil {
		if !isUniqueViolation(err) {
			return entity.Artist{}, false, err
		}
		artists = nil
		if findErr := tx.Unscoped().
			Where("spotify_artist_id = ?", candidate.SpotifyArtistID).
			Limit(1).
			Find(&artists).Error; findErr != nil || len(artists) == 0 {
			if findErr != nil {
				return entity.Artist{}, false, findErr
			}
			return entity.Artist{}, false, err
		}
		return artists[0], false, nil
	}
	return candidate, true, nil
}

func findCandidateTags(tx *gorm.DB, slugs []string) ([]entity.Tag, error) {
	if len(slugs) == 0 {
		return nil, nil
	}
	var tags []entity.Tag
	if err := tx.Where("slug IN ?", slugs).Find(&tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}

func hasRequiredCandidateTags(tags []entity.Tag) bool {
	hasGenre := false
	hasContact := false
	for _, tag := range tags {
		switch {
		case tag.Category == "genre":
			hasGenre = true
		case tag.Category == "mood", tag.Category == "feature", tag.Category == "scene":
			hasContact = true
		}
	}
	return hasGenre && hasContact
}

func upsertAutoApprovedSongTags(tx *gorm.DB, songID uuid.UUID, tags []entity.Tag) error {
	for _, tag := range tags {
		value := entity.SongTag{SongID: songID, TagID: tag.ID, Weight: 1}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "song_id"}, {Name: "tag_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"weight", "updated_at"}),
		}).Create(&value).Error; err != nil {
			return err
		}
	}
	return nil
}

func upsertAutoApprovedArtistTags(tx *gorm.DB, artistID uuid.UUID, tags []entity.Tag) error {
	for _, tag := range tags {
		value := entity.ArtistTag{ArtistID: artistID, TagID: tag.ID}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "artist_id"}, {Name: "tag_id"}},
			DoNothing: true,
		}).Create(&value).Error; err != nil {
			return err
		}
	}
	return nil
}
