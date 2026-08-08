package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"music-discovery/backend/entity"
)

func (r *adminRepository) ListSongs(ctx context.Context, q PageQuery) ([]entity.SongDetail, int64, error) {
	db := addKeyword(r.db.WithContext(ctx).Model(&entity.Song{}), q.Keyword, "songs.name", "songs.album_name", "songs.spotify_track_id")
	if q.ArtistID != nil {
		db = db.Where("songs.artist_id = ?", *q.ArtistID)
	}
	if q.TagID != nil {
		db = db.Joins("JOIN song_tags ON song_tags.song_id = songs.id").Where("song_tags.tag_id = ?", *q.TagID).Distinct("songs.*")
	}
	if q.Explicit != nil {
		db = db.Where("songs.explicit = ?", *q.Explicit)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, repoError(err)
	}
	var songs []entity.Song
	if err := db.Order(safeOrder(q.Sort, q.Order, map[string]string{"name": "songs.name", "popularity": "songs.popularity", "release_date": "songs.release_date", "created_at": "songs.created_at", "updated_at": "songs.updated_at"}, "songs.updated_at")).Offset(offset(q)).Limit(q.Limit).Find(&songs).Error; err != nil {
		return nil, 0, repoError(err)
	}
	items := make([]entity.SongDetail, 0, len(songs))
	for _, song := range songs {
		detail, err := r.songDetail(ctx, song)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *detail)
	}
	return items, total, nil
}
func (r *adminRepository) FindSong(ctx context.Context, id uuid.UUID) (*entity.Song, error) {
	var value entity.Song
	if err := r.db.WithContext(ctx).First(&value, "id = ?", id).Error; err != nil {
		return nil, repoError(err)
	}
	return &value, nil
}
func (r *adminRepository) FindSongDetail(ctx context.Context, id uuid.UUID) (*entity.SongDetail, error) {
	song, err := r.FindSong(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.songDetail(ctx, *song)
}

func (r *adminRepository) songDetail(ctx context.Context, song entity.Song) (*entity.SongDetail, error) {
	var artist entity.Artist
	if err := r.db.WithContext(ctx).Unscoped().First(&artist, "id = ?", song.ArtistID).Error; err != nil {
		return nil, repoError(err)
	}
	var rows []struct {
		entity.Tag
		Weight float64
	}
	if err := r.db.WithContext(ctx).Table("tags").Select("tags.*, song_tags.weight").Joins("JOIN song_tags ON song_tags.tag_id = tags.id").Where("song_tags.song_id = ?", song.ID).Scan(&rows).Error; err != nil {
		return nil, repoError(err)
	}
	tags := make([]entity.SongTagDetail, 0, len(rows))
	for _, row := range rows {
		tags = append(tags, entity.SongTagDetail{Tag: row.Tag, Weight: row.Weight})
	}
	metrics, err := r.songMetrics(ctx, song.ID)
	if err != nil {
		return nil, err
	}
	return &entity.SongDetail{Song: song, Artist: artist, Tags: tags, Metrics: metrics}, nil
}

func (r *adminRepository) songMetrics(ctx context.Context, songID uuid.UUID) (entity.SongMetrics, error) {
	var value entity.SongMetrics
	if r.db.Migrator().HasTable("recommendation_details") {
		if err := r.db.WithContext(ctx).Table("recommendation_details").Where("song_id = ?", songID).Count(&value.RecommendationCount).Error; err != nil {
			return value, repoError(err)
		}
	}
	if r.db.Migrator().HasTable("feedbacks") {
		var rows []struct {
			FeedbackType string
			Count        int64
		}
		if err := r.db.WithContext(ctx).Table("feedbacks").Select("feedback_type, COUNT(*) AS count").Where("song_id = ?", songID).Group("feedback_type").Scan(&rows).Error; err != nil {
			return value, repoError(err)
		}
		for _, row := range rows {
			switch row.FeedbackType {
			case "like":
				value.LikeCount = row.Count
			case "maybe":
				value.MaybeCount = row.Count
			case "skip":
				value.SkipCount = row.Count
			}
		}
	}
	return value, nil
}
func replaceSongTags(tx *gorm.DB, songID uuid.UUID, tags []entity.SongTag, now time.Time) error {
	if err := tx.Where("song_id = ?", songID).Delete(&entity.SongTag{}).Error; err != nil {
		return repoError(err)
	}
	for _, tag := range tags {
		tag.ID = uuid.New()
		tag.SongID = songID
		tag.CreatedAt = now
		tag.UpdatedAt = now
		if err := tx.Create(&tag).Error; err != nil {
			return repoError(err)
		}
	}
	return nil
}
func (r *adminRepository) SaveSongWithTagsAndAudit(ctx context.Context, value *entity.Song, tags []entity.SongTag, a AuditContext) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(value).Error; err != nil {
			return repoError(err)
		}
		if err := replaceSongTags(tx, value.ID, tags, a.CreatedAt); err != nil {
			return err
		}
		a.TargetID = value.ID.String()
		if a.After == nil {
			a.After = value
		}
		return createAudit(tx, a)
	})
}
func (r *adminRepository) DeleteSongWithAudit(ctx context.Context, id uuid.UUID, a AuditContext) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var value entity.Song
		if err := tx.First(&value, "id = ?", id).Error; err != nil {
			return repoError(err)
		}
		result := tx.Delete(&entity.Song{}, "id = ?", id)
		if result.Error != nil {
			return repoError(result.Error)
		}
		if result.RowsAffected == 0 {
			return entity.ErrNotFound
		}
		a.TargetID = id.String()
		a.Before = value
		return createAudit(tx, a)
	})
}
