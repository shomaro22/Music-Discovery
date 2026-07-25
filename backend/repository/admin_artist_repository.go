package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"music-discovery/backend/entity"
)

func (r *adminRepository) ListArtists(ctx context.Context, q PageQuery) ([]ArtistListItem, int64, error) {
	db := addKeyword(r.db.WithContext(ctx).Model(&entity.Artist{}), q.Keyword, "artists.name", "artists.spotify_artist_id")
	if q.TagID != nil {
		db = db.Joins("JOIN artist_tags ON artist_tags.artist_id = artists.id").Where("artist_tags.tag_id = ?", *q.TagID).Distinct("artists.*")
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, repoError(err)
	}
	var artists []entity.Artist
	if err := db.Order(safeOrder(q.Sort, q.Order, map[string]string{"name": "artists.name", "popularity": "artists.popularity", "created_at": "artists.created_at", "updated_at": "artists.updated_at"}, "artists.updated_at")).Offset(offset(q)).Limit(q.Limit).Find(&artists).Error; err != nil {
		return nil, 0, repoError(err)
	}
	items := make([]ArtistListItem, 0, len(artists))
	if len(artists) == 0 {
		return items, total, nil
	}
	ids := make([]uuid.UUID, 0, len(artists))
	for _, artist := range artists {
		ids = append(ids, artist.ID)
	}
	var rows []struct {
		ArtistID uuid.UUID
		entity.Tag
	}
	if err := r.db.WithContext(ctx).Table("tags").Select("artist_tags.artist_id, tags.*").Joins("JOIN artist_tags ON artist_tags.tag_id = tags.id").Where("artist_tags.artist_id IN ? AND tags.deleted_at IS NULL", ids).Order("tags.name ASC").Scan(&rows).Error; err != nil {
		return nil, 0, repoError(err)
	}
	tagsByArtist := make(map[uuid.UUID][]entity.Tag, len(artists))
	for _, row := range rows {
		tagsByArtist[row.ArtistID] = append(tagsByArtist[row.ArtistID], row.Tag)
	}
	for _, artist := range artists {
		items = append(items, ArtistListItem{Artist: artist, Tags: tagsByArtist[artist.ID]})
	}
	return items, total, nil
}
func (r *adminRepository) FindArtist(ctx context.Context, id uuid.UUID) (*entity.Artist, error) {
	var value entity.Artist
	if err := r.db.WithContext(ctx).First(&value, "id = ?", id).Error; err != nil {
		return nil, repoError(err)
	}
	return &value, nil
}
func (r *adminRepository) FindArtistDetail(ctx context.Context, id uuid.UUID) (*ArtistDetail, error) {
	artist, err := r.FindArtist(ctx, id)
	if err != nil {
		return nil, err
	}
	var tags []entity.Tag
	if err = r.db.WithContext(ctx).Table("tags").Joins("JOIN artist_tags ON artist_tags.tag_id = tags.id").Where("artist_tags.artist_id = ? AND tags.deleted_at IS NULL", id).Find(&tags).Error; err != nil {
		return nil, repoError(err)
	}
	var count int64
	if err = r.db.WithContext(ctx).Model(&entity.Song{}).Where("artist_id = ?", id).Count(&count).Error; err != nil {
		return nil, repoError(err)
	}
	return &ArtistDetail{Artist: *artist, Tags: tags, SongCount: count}, nil
}

func replaceArtistTags(tx *gorm.DB, artistID uuid.UUID, tagIDs []uuid.UUID, now time.Time) error {
	if err := tx.Where("artist_id = ?", artistID).Delete(&entity.ArtistTag{}).Error; err != nil {
		return repoError(err)
	}
	for _, id := range tagIDs {
		link := entity.ArtistTag{ID: uuid.New(), ArtistID: artistID, TagID: id, CreatedAt: now}
		if err := tx.Create(&link).Error; err != nil {
			return repoError(err)
		}
	}
	return nil
}
func (r *adminRepository) SaveArtistWithTagsAndAudit(ctx context.Context, value *entity.Artist, tagIDs []uuid.UUID, a AuditContext) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(value).Error; err != nil {
			return repoError(err)
		}
		if err := replaceArtistTags(tx, value.ID, tagIDs, a.CreatedAt); err != nil {
			return err
		}
		a.TargetID = value.ID.String()
		if a.After == nil {
			a.After = value
		}
		return createAudit(tx, a)
	})
}
func (r *adminRepository) DeleteArtistWithAudit(ctx context.Context, id uuid.UUID, a AuditContext) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var value entity.Artist
		if err := tx.First(&value, "id = ?", id).Error; err != nil {
			return repoError(err)
		}
		result := tx.Delete(&entity.Artist{}, "id = ?", id)
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
