package repository

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"music-discovery/backend/entity"
)

type ISpotifyCandidateRepository interface {
	UpsertCandidates(context.Context, []entity.SpotifyCandidate) error
}

type spotifyCandidateRepository struct{ db *gorm.DB }

func NewSpotifyCandidateRepository(db *gorm.DB) ISpotifyCandidateRepository {
	return &spotifyCandidateRepository{db: db}
}

func (r *spotifyCandidateRepository) UpsertCandidates(ctx context.Context, values []entity.SpotifyCandidate) error {
	if len(values) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, value := range values {
			artist := value.Artist
			deleted, err := deletedSpotifyArtistExists(tx, artist.SpotifyArtistID)
			if err != nil {
				return entity.ErrInternal
			}
			if deleted {
				// 管理者が削除したArtistには更新も復元も行わない。
				continue
			}

			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "spotify_artist_id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"name", "spotify_url", "updated_at",
				}),
				Where: clause.Where{Exprs: []clause.Expression{
					clause.Eq{Column: clause.Column{Table: "artists", Name: "deleted_at"}, Value: nil},
				}},
			}).Create(&artist).Error; err != nil {
				return entity.ErrInternal
			}
			if err := tx.Where("spotify_artist_id = ?", artist.SpotifyArtistID).First(&artist).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					// 管理者が削除したArtistは自動取込で復元しない。
					continue
				}
				return entity.ErrInternal
			}

			song := value.Song
			song.ArtistID = artist.ID
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "spotify_track_id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"artist_id", "name", "album_name", "image_url", "spotify_url",
					"preview_url", "duration_ms", "popularity", "release_date", "explicit", "updated_at",
				}),
			}).Create(&song).Error; err != nil {
				return entity.ErrInternal
			}
		}
		return nil
	})
}

func deletedSpotifyArtistExists(tx *gorm.DB, spotifyArtistID string) (bool, error) {
	var artist entity.Artist
	err := tx.Unscoped().
		Select("id", "deleted_at").
		Where("spotify_artist_id = ?", spotifyArtistID).
		First(&artist).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return artist.DeletedAt.Valid, nil
}
