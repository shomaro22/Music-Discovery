package usecase

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"music-discovery/backend/entity"
	"music-discovery/backend/repository"
)

type PageInput struct {
	Keyword, Sort, Order                             string
	Page, Limit                                      int
	TagID, ArtistID                                  string
	Explicit                                         *bool
	Role, Status                                     string
	IncludeDeleted                                   bool
	Action, ActorID, TargetType, TargetID, RequestID string
	Level, Category, Event, UserID, ExternalService  string
	Method, Path, TagCategory                        string
	StatusCode                                       *int
	From, To                                         string
}
type PageResult[T any] struct {
	Items      []T
	Page       int
	Limit      int
	Total      int64
	TotalPages int
}
type AuditInput struct{ IPAddress, UserAgent, RequestID string }
type ArtistInput struct {
	ID                                          uuid.UUID
	SpotifyArtistID, Name, ImageURL, SpotifyURL string
	Popularity                                  int
	TagIDs                                      []uuid.UUID
}
type ArtistListResponse struct {
	entity.Artist
	Tags []entity.Tag
}

type ArtistResponse struct {
	entity.Artist
	Tags      []entity.Tag
	SongCount int64
}
type SongTagInput struct {
	TagID  uuid.UUID
	Weight float64
}
type SongInput struct {
	Song entity.Song
	Tags []SongTagInput
}
type SongTagResponse struct {
	entity.Tag
	Weight float64
}
type SongResponse struct {
	entity.Song
	Artist              entity.Artist
	Tags                []SongTagResponse
	RecommendationCount int64
	LikeCount           int64
	MaybeCount          int64
	SkipCount           int64
}

type TagResponse struct {
	entity.Tag
	ArtistCount     int64
	SongCount       int64
	UserWeightCount int64
}
type TagOptionResponse struct {
	ID       uuid.UUID
	Name     string
	Slug     string
	Category string
}
type UserAdminResponse struct {
	ID              uuid.UUID
	Name            string
	Email           string
	Role            entity.UserRole
	Status          entity.UserStatus
	SuspendedAt     *time.Time
	SuspendedReason string
	SuspendedBy     *uuid.UUID
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}

func newUserAdminResponse(user entity.User) UserAdminResponse {
	response := UserAdminResponse{
		ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role, Status: user.Status,
		SuspendedAt: user.SuspendedAt, SuspendedReason: user.SuspendedReason, SuspendedBy: user.SuspendedBy,
		CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt,
	}
	if user.DeletedAt.Valid {
		deletedAt := user.DeletedAt.Time
		response.DeletedAt = &deletedAt
	}
	return response
}

type UserDetailResponse struct {
	UserAdminResponse
	ProfileStatus       string
	RecommendationCount int64
	FeedbackCount       int64
	LastLoginAt         *time.Time
}
type AuditLogListResponse struct {
	ID           uuid.UUID
	ActorUserID  uuid.UUID
	ActorRole    entity.Role
	ActorName    string
	ActorEmail   string
	RequestID    string
	Action       string
	TargetType   string
	TargetID     string
	TargetName   string
	TargetDetail string
	Summary      string
	IPAddress    string
	UserAgent    string
	CreatedAt    time.Time
}
type SystemLogListResponse struct {
	ID              uuid.UUID
	Environment     string
	Service         string
	Version         string
	Level           string
	Category        string
	Event           string
	ErrorCode       string
	Message         string
	RequestID       string
	UserID          *uuid.UUID
	UserName        string
	UserEmail       string
	Method          string
	Path            string
	StatusCode      int
	ExternalService string
	DurationMs      int64
	IPAddress       string
	UserAgent       string
	CreatedAt       time.Time
}

type UserStatusInput struct {
	Status entity.UserStatus
	Reason string
}
type RoleChangeInput struct {
	Role   entity.Role
	Reason string
}
type RoleChangeResponse struct {
	UserID     uuid.UUID
	BeforeRole entity.Role
	AfterRole  entity.Role
}

type IContentAdminUsecase interface {
	ListArtists(context.Context, PageInput) (PageResult[ArtistListResponse], error)
	GetArtist(context.Context, uuid.UUID) (*ArtistResponse, error)
	SaveArtist(context.Context, uuid.UUID, ArtistInput, AuditInput) (*ArtistResponse, error)
	DeleteArtist(context.Context, uuid.UUID, uuid.UUID, AuditInput) error
	ListSongs(context.Context, PageInput) (PageResult[SongResponse], error)
	GetSong(context.Context, uuid.UUID) (*SongResponse, error)
	SaveSong(context.Context, uuid.UUID, SongInput, AuditInput) (*SongResponse, error)
	DeleteSong(context.Context, uuid.UUID, uuid.UUID, AuditInput) error
	ListTags(context.Context, PageInput) (PageResult[TagResponse], error)
	GetTag(context.Context, uuid.UUID) (*TagResponse, error)
	ListTagOptions(context.Context) ([]TagOptionResponse, error)
	SaveTag(context.Context, uuid.UUID, *entity.Tag, AuditInput) (*entity.Tag, error)
	DeleteTag(context.Context, uuid.UUID, uuid.UUID, AuditInput) error
}

type ISystemAdminUsecase interface {
	ListUsers(context.Context, PageInput) (PageResult[UserAdminResponse], error)
	GetUser(context.Context, uuid.UUID) (*UserDetailResponse, error)
	SetUserStatus(context.Context, uuid.UUID, uuid.UUID, UserStatusInput, AuditInput) error
	ListRoles(context.Context) []entity.Role
	ChangeRole(context.Context, uuid.UUID, uuid.UUID, RoleChangeInput, AuditInput) (*RoleChangeResponse, error)
	ListAuditLogs(context.Context, PageInput) (PageResult[AuditLogListResponse], error)
	GetAuditLog(context.Context, uuid.UUID) (*entity.AuditLog, error)
	ListSystemLogs(context.Context, PageInput) (PageResult[SystemLogListResponse], error)
	GetSystemLog(context.Context, uuid.UUID) (*entity.SystemLog, error)
}

type IAdminManagementUsecase interface {
	IContentAdminUsecase
	ISystemAdminUsecase
}

type adminManagementUsecase struct {
	repo  repository.IAdminRepository
	clock IClock
}

func NewAdminManagementUsecase(repo repository.IAdminRepository, clock IClock) IAdminManagementUsecase {
	return &adminManagementUsecase{repo: repo, clock: clock}
}

var spotifyIDPattern = regexp.MustCompile(`^[A-Za-z0-9]{10,64}$`)
var tagSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var allowedTagCategories = map[string]struct{}{"genre": {}, "mood": {}, "scene": {}, "feature": {}}
var allowedHTTPMethods = map[string]struct{}{"GET": {}, "POST": {}, "PUT": {}, "PATCH": {}, "DELETE": {}, "OPTIONS": {}, "HEAD": {}}

func validSpotifyURL(raw, resource string) bool {
	u, err := url.ParseRequestURI(raw)
	return err == nil && u.Scheme == "https" && u.Host == "open.spotify.com" && strings.HasPrefix(u.Path, "/"+resource+"/")
}
func allowed(value string, values ...string) bool {
	if value == "" {
		return true
	}
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
func validateSort(sort string, values ...string) bool {
	return allowed(sort, append([]string{""}, values...)...)
}

func normalizePage(in PageInput) (PageInput, error) {
	if in.Page == 0 {
		in.Page = 1
	}
	if in.Limit == 0 {
		in.Limit = 20
	}
	if in.Page < 1 || in.Limit < 1 || in.Limit > 100 {
		return in, entity.ErrInvalidInput
	}
	in.Keyword = strings.TrimSpace(in.Keyword)
	if in.Order == "" {
		in.Order = "desc"
	}
	if in.Order != "asc" && in.Order != "desc" {
		return in, entity.ErrInvalidInput
	}
	return in, nil
}
func query(in PageInput) (repository.PageQuery, error) {
	in, err := normalizePage(in)
	if err != nil {
		return repository.PageQuery{}, err
	}
	if !allowed(in.Role, "user", "content_admin", "system_admin") || !allowed(in.Status, "active", "suspended") || !allowed(in.Level, "debug", "info", "warn", "error", "fatal") || !allowed(in.Category, "application", "access", "security", "http", "database", "redis", "auth", "external_service", "recommendation", "job", "transaction") || (in.Method != "" && func() bool { _, ok := allowedHTTPMethods[strings.ToUpper(in.Method)]; return !ok }()) || (in.StatusCode != nil && (*in.StatusCode < 100 || *in.StatusCode > 599)) {
		return repository.PageQuery{}, entity.ErrInvalidInput
	}
	if in.TagCategory != "" {
		if _, ok := allowedTagCategories[in.TagCategory]; !ok {
			return repository.PageQuery{}, entity.ErrInvalidInput
		}
	}
	q := repository.PageQuery{Keyword: in.Keyword, Sort: in.Sort, Order: in.Order, Page: in.Page, Limit: in.Limit, Explicit: in.Explicit, Role: in.Role, Status: in.Status, IncludeDeleted: in.IncludeDeleted, Action: in.Action, TargetType: in.TargetType, TargetID: in.TargetID, RequestID: in.RequestID, Level: in.Level, Category: in.Category, Event: in.Event, StatusCode: in.StatusCode, ExternalService: in.ExternalService, Method: strings.ToUpper(in.Method), Path: in.Path, TagCategory: in.TagCategory}
	if in.TagID != "" {
		id, e := uuid.Parse(in.TagID)
		if e != nil {
			return q, entity.ErrInvalidInput
		}
		q.TagID = &id
	}
	if in.ArtistID != "" {
		id, e := uuid.Parse(in.ArtistID)
		if e != nil {
			return q, entity.ErrInvalidInput
		}
		q.ArtistID = &id
	}

	if in.ActorID != "" {
		id, e := uuid.Parse(in.ActorID)
		if e != nil {
			return q, entity.ErrInvalidInput
		}
		q.ActorID = &id
	}
	if in.UserID != "" {
		id, e := uuid.Parse(in.UserID)
		if e != nil {
			return q, entity.ErrInvalidInput
		}
		q.UserID = &id
	}
	if in.From != "" {
		value, e := time.Parse(time.RFC3339, in.From)
		if e != nil {
			return q, entity.ErrInvalidInput
		}
		q.From = &value
	}
	if in.To != "" {
		value, e := time.Parse(time.RFC3339, in.To)
		if e != nil {
			return q, entity.ErrInvalidInput
		}
		q.To = &value
	}
	if q.From != nil && q.To != nil && q.From.After(*q.To) {
		return repository.PageQuery{}, entity.ErrInvalidInput
	}
	return q, nil
}
func page[T any](items []T, total int64, in PageInput) (PageResult[T], error) {
	in, err := normalizePage(in)
	if err != nil {
		return PageResult[T]{}, err
	}
	return PageResult[T]{Items: items, Page: in.Page, Limit: in.Limit, Total: total, TotalPages: int((total + int64(in.Limit) - 1) / int64(in.Limit))}, nil
}
func (u *adminManagementUsecase) audit(actor uuid.UUID, action, targetType, reason, summary string, meta AuditInput) repository.AuditContext {
	return repository.AuditContext{ActorUserID: actor, RequestID: meta.RequestID, Action: action, TargetType: targetType, Reason: reason, Summary: summary, IPAddress: meta.IPAddress, UserAgent: meta.UserAgent, CreatedAt: u.clock.Now()}
}
func validPopularity(value int) bool { return value >= 0 && value <= 100 }
func uniqueIDs(ids []uuid.UUID) []uuid.UUID {
	seen := map[uuid.UUID]struct{}{}
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}
func tagIDs(tags []entity.Tag) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(tags))
	for _, tag := range tags {
		ids = append(ids, tag.ID)
	}
	return ids
}

func songTagAuditValues(tags []entity.SongTagDetail) []SongTagInput {
	values := make([]SongTagInput, 0, len(tags))
	for _, tag := range tags {
		values = append(values, SongTagInput{TagID: tag.Tag.ID, Weight: tag.Weight})
	}
	return values
}
