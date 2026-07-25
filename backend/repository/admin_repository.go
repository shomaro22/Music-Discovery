package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"music-discovery/backend/entity"
)

type PageQuery struct {
	Keyword         string
	Sort            string
	Order           string
	Page            int
	Limit           int
	TagID           *uuid.UUID
	ArtistID        *uuid.UUID
	Explicit        *bool
	Role            string
	Status          string
	IncludeDeleted  bool
	Action          string
	ActorID         *uuid.UUID
	TargetType      string
	TargetID        string
	RequestID       string
	Level           string
	Category        string
	Event           string
	UserID          *uuid.UUID
	StatusCode      *int
	ExternalService string
	Method          string
	Path            string
	TagCategory     string
	From            *time.Time
	To              *time.Time
}

type ArtistListItem struct {
	Artist entity.Artist
	Tags   []entity.Tag
}

type TagSummary struct {
	Tag             entity.Tag
	ArtistCount     int64
	SongCount       int64
	UserWeightCount int64
}

type SongMetrics struct {
	RecommendationCount int64
	LikeCount           int64
	MaybeCount          int64
	SkipCount           int64
}

type UserMetrics struct {
	ProfileStatus       string
	RecommendationCount int64
	FeedbackCount       int64
	LastLoginAt         *time.Time
}

type ArtistDetail struct {
	Artist    entity.Artist
	Tags      []entity.Tag
	SongCount int64
}

type SongTagDetail struct {
	Tag    entity.Tag
	Weight float64
}

type SongDetail struct {
	Song    entity.Song
	Artist  entity.Artist
	Tags    []SongTagDetail
	Metrics SongMetrics
}

type AuditContext struct {
	ActorUserID uuid.UUID
	Action      string
	TargetType  string
	TargetID    string
	Reason      string
	Summary     string
	Before      interface{}
	After       interface{}
	IPAddress   string
	UserAgent   string
	RequestID   string
	ActorRole   entity.Role
	ActorName   string
	ActorEmail  string
	CreatedAt   time.Time
}

type IContentAdminRepository interface {
	ListArtists(context.Context, PageQuery) ([]ArtistListItem, int64, error)
	FindArtist(context.Context, uuid.UUID) (*entity.Artist, error)
	FindArtistDetail(context.Context, uuid.UUID) (*ArtistDetail, error)
	SaveArtistWithTagsAndAudit(context.Context, *entity.Artist, []uuid.UUID, AuditContext) error
	DeleteArtistWithAudit(context.Context, uuid.UUID, AuditContext) error
	ListSongs(context.Context, PageQuery) ([]SongDetail, int64, error)
	FindSong(context.Context, uuid.UUID) (*entity.Song, error)
	FindSongDetail(context.Context, uuid.UUID) (*SongDetail, error)
	SaveSongWithTagsAndAudit(context.Context, *entity.Song, []entity.SongTag, AuditContext) error
	DeleteSongWithAudit(context.Context, uuid.UUID, AuditContext) error
	ListTags(context.Context, PageQuery) ([]TagSummary, int64, error)
	FindTag(context.Context, uuid.UUID) (*TagSummary, error)
	ListTagOptions(context.Context) ([]entity.Tag, error)
	SaveTagWithAudit(context.Context, *entity.Tag, AuditContext) error
	DeleteTagWithAudit(context.Context, uuid.UUID, AuditContext) error
	ValidateTagIDs(context.Context, []uuid.UUID) error
}

type ISystemAdminRepository interface {
	ListUsers(context.Context, PageQuery) ([]entity.User, int64, error)
	FindUserUnscoped(context.Context, uuid.UUID) (*entity.User, error)
	FindUserMetrics(context.Context, uuid.UUID) (*UserMetrics, error)
	SetUserStatusWithAudit(context.Context, uuid.UUID, uuid.UUID, entity.UserStatus, string, time.Time, AuditContext) error
	ChangeUserRoleWithAudit(context.Context, uuid.UUID, entity.Role, time.Time, AuditContext) error
	ListAuditLogs(context.Context, PageQuery) ([]entity.AuditLog, int64, error)
	FindAuditLog(context.Context, uuid.UUID) (*entity.AuditLog, error)
	ListSystemLogs(context.Context, PageQuery) ([]entity.SystemLog, int64, error)
	FindSystemLog(context.Context, uuid.UUID) (*entity.SystemLog, error)
	CreateSystemLog(context.Context, *entity.SystemLog) error
}

type IAdminRepository interface {
	IContentAdminRepository
	ISystemAdminRepository
}

type adminRepository struct{ db *gorm.DB }

func NewAdminRepository(db *gorm.DB) IAdminRepository { return &adminRepository{db: db} }

func offset(q PageQuery) int { return (q.Page - 1) * q.Limit }
func safeOrder(sort, order string, allowed map[string]string, fallback string) string {
	column := allowed[sort]
	if column == "" {
		column = fallback
	}
	direction := "DESC"
	if strings.EqualFold(order, "asc") {
		direction = "ASC"
	}
	return column + " " + direction
}
func repoError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return entity.ErrNotFound
	}
	text := strings.ToLower(err.Error())
	if strings.Contains(text, "duplicate") || strings.Contains(text, "unique constraint") {
		return entity.ErrConflict
	}
	return entity.ErrInternal
}
func addKeyword(db *gorm.DB, keyword string, columns ...string) *gorm.DB {
	if keyword == "" {
		return db
	}
	parts := make([]string, len(columns))
	args := make([]interface{}, len(columns))
	for i, column := range columns {
		parts[i] = column + " ILIKE ?"
		args[i] = "%" + keyword + "%"
	}
	return db.Where(strings.Join(parts, " OR "), args...)
}
