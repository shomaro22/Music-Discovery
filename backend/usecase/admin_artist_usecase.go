package usecase

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"music-discovery/backend/entity"
)

func (u *adminManagementUsecase) ListArtists(ctx context.Context, in PageInput) (PageResult[ArtistListResponse], error) {
	if !validateSort(in.Sort, "name", "popularity", "created_at", "updated_at") {
		return PageResult[ArtistListResponse]{}, entity.ErrInvalidInput
	}
	q, e := query(in)
	if e != nil {
		return PageResult[ArtistListResponse]{}, e
	}
	items, total, e := u.repo.ListArtists(ctx, q)
	if e != nil {
		return PageResult[ArtistListResponse]{}, e
	}
	responses := make([]ArtistListResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, ArtistListResponse{Artist: item.Artist, Tags: item.Tags})
	}
	return page(responses, total, in)
}
func (u *adminManagementUsecase) GetArtist(ctx context.Context, id uuid.UUID) (*ArtistResponse, error) {
	if id == uuid.Nil {
		return nil, entity.ErrInvalidInput
	}
	detail, e := u.repo.FindArtistDetail(ctx, id)
	if e != nil {
		return nil, e
	}
	return &ArtistResponse{Artist: detail.Artist, Tags: detail.Tags, SongCount: detail.SongCount}, nil
}
func (u *adminManagementUsecase) SaveArtist(ctx context.Context, actor uuid.UUID, in ArtistInput, meta AuditInput) (*ArtistResponse, error) {
	if actor == uuid.Nil || !spotifyIDPattern.MatchString(strings.TrimSpace(in.SpotifyArtistID)) || len(strings.TrimSpace(in.Name)) > 200 || strings.TrimSpace(in.Name) == "" || !validSpotifyURL(strings.TrimSpace(in.SpotifyURL), "artist") || !validPopularity(in.Popularity) {
		return nil, entity.ErrInvalidInput
	}
	for _, id := range in.TagIDs {
		if id == uuid.Nil {
			return nil, entity.ErrInvalidInput
		}
	}
	ids := uniqueIDs(in.TagIDs)
	if e := u.repo.ValidateTagIDs(ctx, ids); e != nil {
		return nil, e
	}
	if in.ID == uuid.Nil {
		if existing, e := u.repo.FindArtistBySpotifyID(ctx, strings.TrimSpace(in.SpotifyArtistID)); e == nil && existing != nil {
			return nil, entity.ErrArtistAlreadyExists
		} else if e != nil && e != entity.ErrNotFound {
			return nil, e
		}
	}
	value := entity.Artist{ID: in.ID, SpotifyArtistID: strings.TrimSpace(in.SpotifyArtistID), Name: strings.TrimSpace(in.Name), ImageURL: strings.TrimSpace(in.ImageURL), SpotifyURL: strings.TrimSpace(in.SpotifyURL), Popularity: in.Popularity}
	action := "artist_created"
	var before interface{}
	if value.ID == uuid.Nil {
		value.ID = uuid.New()
	} else {
		old, e := u.repo.FindArtistDetail(ctx, value.ID)
		if e != nil {
			return nil, e
		}
		value.CreatedAt = old.Artist.CreatedAt
		before = map[string]interface{}{"artist": old.Artist, "tag_ids": tagIDs(old.Tags)}
		action = "artist_updated"
	}
	audit := u.audit(actor, action, "artist", "", "ArtistとTag関連を保存しました", meta)
	audit.Before = before
	audit.After = map[string]interface{}{"artist": value, "tag_ids": ids}
	if e := u.repo.SaveArtistWithTagsAndAudit(ctx, &value, ids, audit); e != nil {
		return nil, e
	}
	return u.GetArtist(ctx, value.ID)
}
func (u *adminManagementUsecase) DeleteArtist(ctx context.Context, actor, id uuid.UUID, meta AuditInput) error {
	if actor == uuid.Nil || id == uuid.Nil {
		return entity.ErrInvalidInput
	}
	return u.repo.DeleteArtistWithAudit(ctx, id, u.audit(actor, "artist_deleted", "artist", "", "Artistを論理削除しました", meta))
}
