package repository

import (
	"context"
	"strings"

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

// ICuratedCatalogRepository persists the trusted, Japan-oriented foundation catalog.
// It is intentionally separate from automatic Spotify candidate import.
type ICuratedCatalogRepository interface {
	FindApprovedSongKeys(context.Context) (map[string]bool, error)
	UpsertApprovedCandidates(context.Context, []entity.SpotifyCandidate) (int, error)
}

type spotifyCandidateRepository struct{ db *gorm.DB }

func NewSpotifyCandidateRepository(db *gorm.DB) ISpotifyCandidateRepository {
	return &spotifyCandidateRepository{db: db}
}

func NewCuratedCatalogRepository(db *gorm.DB) ICuratedCatalogRepository {
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

func (r *spotifyCandidateRepository) FindApprovedSongKeys(ctx context.Context) (map[string]bool, error) {
	type row struct {
		ArtistName string
		SongName   string
	}
	var rows []row
	if err := r.db.WithContext(ctx).
		Table("songs").
		Select("artists.name AS artist_name, songs.name AS song_name").
		Joins("JOIN artists ON artists.id = songs.artist_id").
		Where("songs.deleted_at IS NULL").
		Where("artists.deleted_at IS NULL").
		Where("songs.review_status = ?", entity.ContentReviewStatusApproved).
		Scan(&rows).Error; err != nil {
		return nil, entity.ErrInternal
	}
	result := make(map[string]bool, len(rows))
	for _, value := range rows {
		result[curatedSongKey(value.ArtistName, value.SongName)] = true
	}
	return result, nil
}

func (r *spotifyCandidateRepository) UpsertApprovedCandidates(
	ctx context.Context,
	values []entity.SpotifyCandidate,
) (int, error) {
	if len(values) == 0 {
		return 0, nil
	}
	count := 0
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, value := range values {
			if err := upsertApprovedCandidate(tx, value); err != nil {
				return err
			}
			count++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

func upsertApprovedCandidate(tx *gorm.DB, value entity.SpotifyCandidate) error {
	artist, _, err := findOrCreateSpotifyArtist(tx, value.Artist)
	if err != nil {
		return entity.ErrInternal
	}
	if artist.DeletedAt.Valid {
		if err := tx.Unscoped().Model(&artist).Updates(map[string]interface{}{
			"deleted_at":  nil,
			"name":        value.Artist.Name,
			"image_url":   value.Artist.ImageURL,
			"spotify_url": value.Artist.SpotifyURL,
			"popularity":  value.Artist.Popularity,
		}).Error; err != nil {
			return entity.ErrInternal
		}
	}

	song := value.Song
	song.ArtistID = artist.ID
	song.ReviewStatus = entity.ContentReviewStatusApproved
	if err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "spotify_track_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"artist_id":     song.ArtistID,
			"name":          song.Name,
			"album_name":    song.AlbumName,
			"image_url":     song.ImageURL,
			"spotify_url":   song.SpotifyURL,
			"preview_url":   song.PreviewURL,
			"duration_ms":   song.DurationMs,
			"popularity":    song.Popularity,
			"release_date":  song.ReleaseDate,
			"explicit":      song.Explicit,
			"review_status": entity.ContentReviewStatusApproved,
			"deleted_at":    nil,
		}),
	}).Create(&song).Error; err != nil {
		return entity.ErrInternal
	}
	if err := tx.Unscoped().Where("spotify_track_id = ?", song.SpotifyTrackID).First(&song).Error; err != nil {
		return entity.ErrInternal
	}

	tags, err := findCandidateTags(tx, value.TagSlugs)
	if err != nil || !hasRequiredCandidateTags(tags) {
		return entity.ErrInternal
	}
	if err := upsertAutoApprovedSongTags(tx, song.ID, tags); err != nil {
		return entity.ErrInternal
	}
	if err := upsertAutoApprovedArtistTags(tx, artist.ID, tags); err != nil {
		return entity.ErrInternal
	}
	return nil
}

func curatedSongKey(artist, song string) string {
	return normalizeCuratedKey(artist) + "|" + normalizeCuratedKey(song)
}

func normalizeCuratedKey(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "　", " ")
	value = strings.ReplaceAll(value, "’", "'")
	return strings.Join(strings.Fields(value), " ")
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
