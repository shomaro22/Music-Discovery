package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"music-discovery/backend/entity"
)

func (r *adminRepository) ListTags(ctx context.Context, q PageQuery) ([]entity.TagSummary, int64, error) {
	db := addKeyword(r.db.WithContext(ctx).Model(&entity.Tag{}), q.Keyword, "name", "slug", "category")
	if q.TagCategory != "" {
		db = db.Where("category = ?", q.TagCategory)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, repoError(err)
	}
	var tags []entity.Tag
	if err := db.Order(safeOrder(q.Sort, q.Order, map[string]string{"name": "name", "category": "category", "created_at": "created_at", "updated_at": "updated_at"}, "updated_at")).Offset(offset(q)).Limit(q.Limit).Find(&tags).Error; err != nil {
		return nil, 0, repoError(err)
	}
	items := make([]entity.TagSummary, 0, len(tags))
	for _, tag := range tags {
		value, err := r.tagSummary(ctx, tag)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *value)
	}
	return items, total, nil
}

func (r *adminRepository) FindTag(ctx context.Context, id uuid.UUID) (*entity.TagSummary, error) {
	var tag entity.Tag
	if err := r.db.WithContext(ctx).First(&tag, "id = ?", id).Error; err != nil {
		return nil, repoError(err)
	}
	return r.tagSummary(ctx, tag)
}

func (r *adminRepository) tagSummary(ctx context.Context, tag entity.Tag) (*entity.TagSummary, error) {
	value := &entity.TagSummary{Tag: tag}
	if err := r.db.WithContext(ctx).Model(&entity.ArtistTag{}).Where("tag_id = ?", tag.ID).Count(&value.ArtistCount).Error; err != nil {
		return nil, repoError(err)
	}
	if err := r.db.WithContext(ctx).Model(&entity.SongTag{}).Where("tag_id = ?", tag.ID).Count(&value.SongCount).Error; err != nil {
		return nil, repoError(err)
	}
	if r.db.Migrator().HasTable("user_tag_weights") {
		if err := r.db.WithContext(ctx).Table("user_tag_weights").Where("tag_id = ?", tag.ID).Count(&value.UserWeightCount).Error; err != nil {
			return nil, repoError(err)
		}
	}
	return value, nil
}

func (r *adminRepository) ListTagOptions(ctx context.Context) ([]entity.Tag, error) {
	var items []entity.Tag
	if err := r.db.WithContext(ctx).Select("id", "name", "slug", "category").Order("category ASC, name ASC").Find(&items).Error; err != nil {
		return nil, repoError(err)
	}
	return items, nil
}

func (r *adminRepository) SaveTagWithAudit(ctx context.Context, value *entity.Tag, a AuditContext) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(value).Error; err != nil {
			return repoError(err)
		}
		a.TargetID = value.ID.String()
		if a.After == nil {
			a.After = value
		}
		return createAudit(tx, a)
	})
}
func (r *adminRepository) DeleteTagWithAudit(ctx context.Context, id uuid.UUID, a AuditContext) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var value entity.Tag
		if err := tx.First(&value, "id = ?", id).Error; err != nil {
			return repoError(err)
		}
		result := tx.Delete(&entity.Tag{}, "id = ?", id)
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
func (r *adminRepository) ValidateTagIDs(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	var count int64
	if err := r.db.WithContext(ctx).Model(&entity.Tag{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return repoError(err)
	}
	if count != int64(len(ids)) {
		return entity.ErrInvalidInput
	}
	return nil
}
