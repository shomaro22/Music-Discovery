package usecase

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"music-discovery/backend/entity"
)

func (u *adminManagementUsecase) ListSongs(ctx context.Context, in PageInput) (PageResult[SongResponse], error) {
	if !validateSort(in.Sort, "name", "popularity", "release_date", "created_at", "updated_at") {
		return PageResult[SongResponse]{}, entity.ErrInvalidInput
	}
	q, e := query(in)
	if e != nil {
		return PageResult[SongResponse]{}, e
	}
	items, total, e := u.repo.ListSongs(ctx, q)
	if e != nil {
		return PageResult[SongResponse]{}, e
	}
	responses := make([]SongResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, songResponse(item))
	}
	return page(responses, total, in)
}
func (u *adminManagementUsecase) GetSong(ctx context.Context, id uuid.UUID) (*SongResponse, error) {
	if id == uuid.Nil {
		return nil, entity.ErrInvalidInput
	}
	detail, e := u.repo.FindSongDetail(ctx, id)
	if e != nil {
		return nil, e
	}
	tags := make([]SongTagResponse, 0, len(detail.Tags))
	for _, tag := range detail.Tags {
		tags = append(tags, SongTagResponse{Tag: tag.Tag, Weight: tag.Weight})
	}
	return &SongResponse{Song: detail.Song, Artist: detail.Artist, Tags: tags, RecommendationCount: detail.Metrics.RecommendationCount, LikeCount: detail.Metrics.LikeCount, MaybeCount: detail.Metrics.MaybeCount, SkipCount: detail.Metrics.SkipCount}, nil
}
func (u *adminManagementUsecase) SaveSong(ctx context.Context, actor uuid.UUID, in SongInput, meta AuditInput) (*SongResponse, error) {
	value := in.Song
	// コンテンツ管理者が保存したSongは、Tag確認済みとして扱う。
	value.ReviewStatus = entity.ContentReviewStatusApproved
	if actor == uuid.Nil || value.ArtistID == uuid.Nil || !spotifyIDPattern.MatchString(strings.TrimSpace(value.SpotifyTrackID)) || strings.TrimSpace(value.Name) == "" || len(strings.TrimSpace(value.Name)) > 200 || strings.TrimSpace(value.AlbumName) == "" || len(strings.TrimSpace(value.AlbumName)) > 200 || !validSpotifyURL(strings.TrimSpace(value.SpotifyURL), "track") || value.DurationMs <= 0 || !validPopularity(value.Popularity) || value.ReleaseDate.IsZero() {
		return nil, entity.ErrInvalidInput
	}
	if _, e := u.repo.FindArtist(ctx, value.ArtistID); e != nil {
		return nil, entity.ErrInvalidInput
	}
	ids := make([]uuid.UUID, 0, len(in.Tags))
	tags := make([]entity.SongTag, 0, len(in.Tags))
	seen := map[uuid.UUID]struct{}{}
	for _, item := range in.Tags {
		if item.TagID == uuid.Nil || item.Weight < 0 || item.Weight > 1 {
			return nil, entity.ErrInvalidInput
		}
		if _, ok := seen[item.TagID]; ok {
			return nil, entity.ErrInvalidInput
		}
		seen[item.TagID] = struct{}{}
		ids = append(ids, item.TagID)
		tags = append(tags, entity.SongTag{TagID: item.TagID, Weight: item.Weight})
	}
	if e := u.repo.ValidateTagIDs(ctx, ids); e != nil {
		return nil, e
	}
	action := "song_created"
	var before interface{}
	if value.ID == uuid.Nil {
		value.ID = uuid.New()
	} else {
		old, e := u.repo.FindSongDetail(ctx, value.ID)
		if e != nil {
			return nil, e
		}
		value.CreatedAt = old.Song.CreatedAt
		before = map[string]interface{}{"song": old.Song, "tags": songTagAuditValues(old.Tags)}
		action = "song_updated"
	}
	audit := u.audit(actor, action, "song", "", "SongとTag Weightを保存しました", meta)
	audit.Before = before
	audit.After = map[string]interface{}{"song": value, "tags": in.Tags}
	if e := u.repo.SaveSongWithTagsAndAudit(ctx, &value, tags, audit); e != nil {
		return nil, e
	}
	return u.GetSong(ctx, value.ID)
}
func (u *adminManagementUsecase) DeleteSong(ctx context.Context, actor, id uuid.UUID, meta AuditInput) error {
	if actor == uuid.Nil || id == uuid.Nil {
		return entity.ErrInvalidInput
	}
	return u.repo.DeleteSongWithAudit(ctx, id, u.audit(actor, "song_deleted", "song", "", "Songを論理削除しました", meta))
}

func songResponse(detail entity.SongDetail) SongResponse {
	tags := make([]SongTagResponse, 0, len(detail.Tags))
	for _, tag := range detail.Tags {
		tags = append(tags, SongTagResponse{Tag: tag.Tag, Weight: tag.Weight})
	}
	return SongResponse{Song: detail.Song, Artist: detail.Artist, Tags: tags, RecommendationCount: detail.Metrics.RecommendationCount, LikeCount: detail.Metrics.LikeCount, MaybeCount: detail.Metrics.MaybeCount, SkipCount: detail.Metrics.SkipCount}
}
