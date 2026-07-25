package usecase

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"music-discovery/backend/entity"
)

func (u *adminManagementUsecase) ListTags(ctx context.Context, in PageInput) (PageResult[TagResponse], error) {
	if !validateSort(in.Sort, "name", "category", "created_at", "updated_at") {
		return PageResult[TagResponse]{}, entity.ErrInvalidInput
	}
	q, e := query(in)
	if e != nil {
		return PageResult[TagResponse]{}, e
	}
	items, total, e := u.repo.ListTags(ctx, q)
	if e != nil {
		return PageResult[TagResponse]{}, e
	}
	responses := make([]TagResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, TagResponse{Tag: item.Tag, ArtistCount: item.ArtistCount, SongCount: item.SongCount, UserWeightCount: item.UserWeightCount})
	}
	return page(responses, total, in)
}
func (u *adminManagementUsecase) GetTag(ctx context.Context, id uuid.UUID) (*TagResponse, error) {
	if id == uuid.Nil {
		return nil, entity.ErrInvalidInput
	}
	value, err := u.repo.FindTag(ctx, id)
	if err != nil {
		return nil, err
	}
	return &TagResponse{Tag: value.Tag, ArtistCount: value.ArtistCount, SongCount: value.SongCount, UserWeightCount: value.UserWeightCount}, nil
}

func (u *adminManagementUsecase) ListTagOptions(ctx context.Context) ([]TagOptionResponse, error) {
	items, err := u.repo.ListTagOptions(ctx)
	if err != nil {
		return nil, err
	}
	values := make([]TagOptionResponse, 0, len(items))
	for _, item := range items {
		values = append(values, TagOptionResponse{ID: item.ID, Name: item.Name, Slug: item.Slug, Category: item.Category})
	}
	return values, nil
}
func (u *adminManagementUsecase) SaveTag(ctx context.Context, actor uuid.UUID, value *entity.Tag, meta AuditInput) (*entity.Tag, error) {
	if actor == uuid.Nil || value == nil || strings.TrimSpace(value.Name) == "" || len(strings.TrimSpace(value.Name)) > 100 || !tagSlugPattern.MatchString(strings.TrimSpace(value.Slug)) || len(strings.TrimSpace(value.Slug)) > 100 || len(strings.TrimSpace(value.Description)) > 500 {
		return nil, entity.ErrInvalidInput
	}
	if _, ok := allowedTagCategories[strings.TrimSpace(value.Category)]; !ok {
		return nil, entity.ErrInvalidInput
	}
	action := "tag_created"
	var before interface{}
	if value.ID == uuid.Nil {
		value.ID = uuid.New()
	} else {
		old, e := u.repo.FindTag(ctx, value.ID)
		if e != nil {
			return nil, e
		}
		value.CreatedAt = old.Tag.CreatedAt
		before = old.Tag
		action = "tag_updated"
	}
	audit := u.audit(actor, action, "tag", "", "Tagを保存しました", meta)
	audit.Before = before
	if e := u.repo.SaveTagWithAudit(ctx, value, audit); e != nil {
		return nil, e
	}
	saved, err := u.repo.FindTag(ctx, value.ID)
	if err != nil {
		return nil, err
	}
	return &saved.Tag, nil
}
func (u *adminManagementUsecase) DeleteTag(ctx context.Context, actor, id uuid.UUID, meta AuditInput) error {
	if actor == uuid.Nil || id == uuid.Nil {
		return entity.ErrInvalidInput
	}
	return u.repo.DeleteTagWithAudit(ctx, id, u.audit(actor, "tag_deleted", "tag", "", "Tagを論理削除しました", meta))
}
