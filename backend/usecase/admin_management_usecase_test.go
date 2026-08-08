package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"music-discovery/backend/entity"
	"music-discovery/backend/repository"
)

type adminRepoStub struct {
	user         *entity.User
	statusErr    error
	roleErr      error
	statusCalled bool
	roleCalled   bool
	lastQuery    repository.PageQuery
}

func (*adminRepoStub) ListArtists(context.Context, repository.PageQuery) ([]entity.ArtistListItem, int64, error) {
	return nil, 0, nil
}
func (*adminRepoStub) FindArtist(context.Context, uuid.UUID) (*entity.Artist, error) {
	return nil, entity.ErrNotFound
}
func (*adminRepoStub) FindArtistBySpotifyID(context.Context, string) (*entity.Artist, error) {
	return nil, entity.ErrNotFound
}
func (*adminRepoStub) FindArtistDetail(context.Context, uuid.UUID) (*entity.ArtistDetail, error) {
	return nil, entity.ErrNotFound
}
func (*adminRepoStub) SaveArtistWithTagsAndAudit(context.Context, *entity.Artist, []uuid.UUID, repository.AuditContext) error {
	return nil
}
func (*adminRepoStub) DeleteArtistWithAudit(context.Context, uuid.UUID, repository.AuditContext) error {
	return nil
}
func (*adminRepoStub) ListSongs(context.Context, repository.PageQuery) ([]entity.SongDetail, int64, error) {
	return nil, 0, nil
}
func (*adminRepoStub) FindSong(context.Context, uuid.UUID) (*entity.Song, error) {
	return nil, entity.ErrNotFound
}
func (*adminRepoStub) FindSongDetail(context.Context, uuid.UUID) (*entity.SongDetail, error) {
	return nil, entity.ErrNotFound
}
func (*adminRepoStub) SaveSongWithTagsAndAudit(context.Context, *entity.Song, []entity.SongTag, repository.AuditContext) error {
	return nil
}
func (*adminRepoStub) DeleteSongWithAudit(context.Context, uuid.UUID, repository.AuditContext) error {
	return nil
}
func (*adminRepoStub) ListTags(context.Context, repository.PageQuery) ([]entity.TagSummary, int64, error) {
	return nil, 0, nil
}
func (*adminRepoStub) FindTag(context.Context, uuid.UUID) (*entity.TagSummary, error) {
	return nil, entity.ErrNotFound
}
func (*adminRepoStub) ListTagOptions(context.Context) ([]entity.Tag, error) {
	return nil, nil
}
func (*adminRepoStub) SaveTagWithAudit(context.Context, *entity.Tag, repository.AuditContext) error {
	return nil
}
func (*adminRepoStub) DeleteTagWithAudit(context.Context, uuid.UUID, repository.AuditContext) error {
	return nil
}
func (*adminRepoStub) ValidateTagIDs(context.Context, []uuid.UUID) error { return nil }
func (s *adminRepoStub) ListUsers(_ context.Context, q repository.PageQuery) ([]entity.User, int64, error) {
	s.lastQuery = q
	return nil, 0, nil
}
func (*adminRepoStub) FindUserMetrics(context.Context, uuid.UUID) (*entity.UserMetrics, error) {
	return &entity.UserMetrics{}, nil
}
func (s *adminRepoStub) FindUserUnscoped(context.Context, uuid.UUID) (*entity.User, error) {
	if s.user == nil {
		return nil, entity.ErrUserNotFound
	}
	copy := *s.user
	return &copy, nil
}
func (s *adminRepoStub) SetUserStatusWithAudit(context.Context, uuid.UUID, uuid.UUID, entity.UserStatus, string, time.Time, repository.AuditContext) error {
	s.statusCalled = true
	return s.statusErr
}
func (s *adminRepoStub) ChangeUserRoleWithAudit(context.Context, uuid.UUID, entity.Role, time.Time, repository.AuditContext) error {
	s.roleCalled = true
	return s.roleErr
}
func (s *adminRepoStub) ListAuditLogs(_ context.Context, q repository.PageQuery) ([]entity.AuditLog, int64, error) {
	s.lastQuery = q
	return nil, 0, nil
}
func (*adminRepoStub) FindAuditLog(context.Context, uuid.UUID) (*entity.AuditLog, error) {
	return nil, entity.ErrNotFound
}
func (*adminRepoStub) ResolveAuditTarget(context.Context, string, string) (string, string) {
	return "", ""
}
func (s *adminRepoStub) ListSystemLogs(_ context.Context, q repository.PageQuery) ([]entity.SystemLog, int64, error) {
	s.lastQuery = q
	return nil, 0, nil
}
func (*adminRepoStub) FindSystemLog(context.Context, uuid.UUID) (*entity.SystemLog, error) {
	return nil, entity.ErrNotFound
}
func (*adminRepoStub) CreateSystemLog(context.Context, *entity.SystemLog) error { return nil }

type adminClock struct{ now time.Time }

func (c adminClock) Now() time.Time { return c.now }

func TestSetUserStatusRequiresSuspensionReason(t *testing.T) {
	repo := &adminRepoStub{}
	usecase := NewAdminManagementUsecase(repo, adminClock{now: time.Now()})
	err := usecase.SetUserStatus(context.Background(), uuid.New(), uuid.New(), UserStatusInput{Status: entity.UserStatusSuspended}, AuditInput{})
	if !errors.Is(err, entity.ErrReasonRequired) {
		t.Fatalf("expected reason required, got %v", err)
	}
	if repo.statusCalled {
		t.Fatal("repository must not be called")
	}
}
func TestChangeRoleRequiresReason(t *testing.T) {
	repo := &adminRepoStub{}
	usecase := NewAdminManagementUsecase(repo, adminClock{now: time.Now()})
	_, err := usecase.ChangeRole(context.Background(), uuid.New(), uuid.New(), RoleChangeInput{Role: entity.RoleUser}, AuditInput{})
	if !errors.Is(err, entity.ErrReasonRequired) {
		t.Fatalf("expected reason required, got %v", err)
	}
	if repo.roleCalled {
		t.Fatal("repository must not be called")
	}
}
func TestChangeRoleReturnsBeforeAndAfterRole(t *testing.T) {
	targetID := uuid.New()
	repo := &adminRepoStub{user: &entity.User{ID: targetID, Role: entity.RoleUser, Status: entity.UserStatusActive}}
	usecase := NewAdminManagementUsecase(repo, adminClock{now: time.Now()})
	result, err := usecase.ChangeRole(context.Background(), uuid.New(), targetID, RoleChangeInput{Role: entity.RoleContentAdmin, Reason: "担当変更"}, AuditInput{})
	if err != nil {
		t.Fatal(err)
	}
	if !repo.roleCalled {
		t.Fatal("repository was not called")
	}
	if result.BeforeRole != entity.RoleUser || result.AfterRole != entity.RoleContentAdmin {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestListUsersPassesRoleStatusAndDeletedFilter(t *testing.T) {
	repo := &adminRepoStub{}
	uc := NewAdminManagementUsecase(repo, adminClock{now: time.Now()})
	_, err := uc.ListUsers(context.Background(), PageInput{Role: "system_admin", Status: "active", IncludeDeleted: true})
	if err != nil {
		t.Fatal(err)
	}
	if repo.lastQuery.Role != "system_admin" || repo.lastQuery.Status != "active" || !repo.lastQuery.IncludeDeleted {
		t.Fatalf("filters not passed: %+v", repo.lastQuery)
	}
}
func TestListAuditLogsPassesDetailedFilters(t *testing.T) {
	actor := uuid.New()
	repo := &adminRepoStub{}
	uc := NewAdminManagementUsecase(repo, adminClock{now: time.Now()})
	_, err := uc.ListAuditLogs(context.Background(), PageInput{Action: "artist_updated", ActorID: actor.String(), TargetType: "artist", RequestID: "req-1", From: "2026-07-01T00:00:00Z", To: "2026-08-01T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	if repo.lastQuery.ActorID == nil || *repo.lastQuery.ActorID != actor || repo.lastQuery.Action != "artist_updated" || repo.lastQuery.RequestID != "req-1" {
		t.Fatalf("filters not passed: %+v", repo.lastQuery)
	}
}
func TestListSystemLogsPassesDetailedFilters(t *testing.T) {
	user := uuid.New()
	status := 429
	repo := &adminRepoStub{}
	uc := NewAdminManagementUsecase(repo, adminClock{now: time.Now()})
	_, err := uc.ListSystemLogs(context.Background(), PageInput{Level: "warn", Category: "http", Event: "rate_limit_exceeded", UserID: user.String(), StatusCode: &status, ExternalService: "spotify"})
	if err != nil {
		t.Fatal(err)
	}
	if repo.lastQuery.UserID == nil || *repo.lastQuery.UserID != user || repo.lastQuery.StatusCode == nil || *repo.lastQuery.StatusCode != 429 || repo.lastQuery.Event != "rate_limit_exceeded" {
		t.Fatalf("filters not passed: %+v", repo.lastQuery)
	}
}
