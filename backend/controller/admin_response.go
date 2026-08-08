package controller

import (
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/usecase"

	"github.com/google/uuid"
)

type adminPageResponse[T any] struct {
	Items      []T   `json:"items"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

func mapAdminPage[S any, T any](value usecase.PageResult[S], convert func(S) T) adminPageResponse[T] {
	items := make([]T, 0, len(value.Items))
	for _, item := range value.Items {
		items = append(items, convert(item))
	}
	return adminPageResponse[T]{Items: items, Page: value.Page, Limit: value.Limit, Total: value.Total, TotalPages: value.TotalPages}
}

type artistListAdminResponse struct {
	entity.Artist
	Tags []entity.Tag `json:"tags"`
}

type artistAdminResponse struct {
	entity.Artist
	Tags      []entity.Tag `json:"tags"`
	SongCount int64        `json:"song_count,omitempty"`
}

func toArtistListAdminResponse(value usecase.ArtistListResponse) artistListAdminResponse {
	return artistListAdminResponse{Artist: value.Artist, Tags: value.Tags}
}

func toArtistAdminResponse(value usecase.ArtistResponse) artistAdminResponse {
	return artistAdminResponse{Artist: value.Artist, Tags: value.Tags, SongCount: value.SongCount}
}

type songTagAdminResponse struct {
	entity.Tag
	Weight float64 `json:"weight"`
}

type songAdminResponse struct {
	entity.Song
	Artist              entity.Artist          `json:"artist"`
	Tags                []songTagAdminResponse `json:"tags"`
	RecommendationCount int64                  `json:"recommendation_count"`
	LikeCount           int64                  `json:"like_count"`
	MaybeCount          int64                  `json:"maybe_count"`
	SkipCount           int64                  `json:"skip_count"`
}

func toSongAdminResponse(value usecase.SongResponse) songAdminResponse {
	tags := make([]songTagAdminResponse, 0, len(value.Tags))
	for _, tag := range value.Tags {
		tags = append(tags, songTagAdminResponse{Tag: tag.Tag, Weight: tag.Weight})
	}
	return songAdminResponse{
		Song: value.Song, Artist: value.Artist, Tags: tags,
		RecommendationCount: value.RecommendationCount, LikeCount: value.LikeCount,
		MaybeCount: value.MaybeCount, SkipCount: value.SkipCount,
	}
}

type tagAdminResponse struct {
	entity.Tag
	ArtistCount     int64 `json:"artist_count"`
	SongCount       int64 `json:"song_count"`
	UserWeightCount int64 `json:"user_weight_count"`
}

type tagOptionAdminResponse struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Slug     string    `json:"slug"`
	Category string    `json:"category"`
}

func toTagAdminResponse(value usecase.TagResponse) tagAdminResponse {
	return tagAdminResponse{Tag: value.Tag, ArtistCount: value.ArtistCount, SongCount: value.SongCount, UserWeightCount: value.UserWeightCount}
}

func toTagOptionAdminResponse(value usecase.TagOptionResponse) tagOptionAdminResponse {
	return tagOptionAdminResponse{ID: value.ID, Name: value.Name, Slug: value.Slug, Category: value.Category}
}

type userAdminResponseDTO struct {
	ID              uuid.UUID         `json:"id"`
	Name            string            `json:"name"`
	Email           string            `json:"email"`
	Role            entity.UserRole   `json:"role"`
	Status          entity.UserStatus `json:"status"`
	SuspendedAt     *time.Time        `json:"suspended_at,omitempty"`
	SuspendedReason string            `json:"suspended_reason,omitempty"`
	SuspendedBy     *uuid.UUID        `json:"suspended_by,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	DeletedAt       *time.Time        `json:"deleted_at,omitempty"`
}

func toUserAdminResponse(value usecase.UserAdminResponse) userAdminResponseDTO {
	return userAdminResponseDTO{
		ID: value.ID, Name: value.Name, Email: value.Email, Role: value.Role, Status: value.Status,
		SuspendedAt: value.SuspendedAt, SuspendedReason: value.SuspendedReason, SuspendedBy: value.SuspendedBy,
		CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt, DeletedAt: value.DeletedAt,
	}
}

type userDetailAdminResponseDTO struct {
	userAdminResponseDTO
	ProfileStatus       string     `json:"profile_status"`
	RecommendationCount int64      `json:"recommendation_count"`
	FeedbackCount       int64      `json:"feedback_count"`
	LastLoginAt         *time.Time `json:"last_login_at,omitempty"`
}

func toUserDetailAdminResponse(value usecase.UserDetailResponse) userDetailAdminResponseDTO {
	return userDetailAdminResponseDTO{
		userAdminResponseDTO: toUserAdminResponse(value.UserAdminResponse),
		ProfileStatus:        value.ProfileStatus, RecommendationCount: value.RecommendationCount,
		FeedbackCount: value.FeedbackCount, LastLoginAt: value.LastLoginAt,
	}
}

type auditLogListAdminResponse struct {
	ID           uuid.UUID   `json:"id"`
	ActorUserID  uuid.UUID   `json:"actor_user_id"`
	ActorRole    entity.Role `json:"actor_role"`
	ActorName    string      `json:"actor_name"`
	ActorEmail   string      `json:"actor_email"`
	RequestID    string      `json:"request_id"`
	Action       string      `json:"action"`
	TargetType   string      `json:"target_type"`
	TargetID     string      `json:"target_id"`
	TargetName   string      `json:"target_name"`
	TargetDetail string      `json:"target_detail"`
	Summary      string      `json:"summary"`
	IPAddress    string      `json:"ip_address"`
	UserAgent    string      `json:"user_agent"`
	CreatedAt    time.Time   `json:"created_at"`
}

func toAuditLogListAdminResponse(value usecase.AuditLogListResponse) auditLogListAdminResponse {
	return auditLogListAdminResponse{
		ID: value.ID, ActorUserID: value.ActorUserID, ActorRole: value.ActorRole, ActorName: value.ActorName,
		ActorEmail: value.ActorEmail, RequestID: value.RequestID, Action: value.Action, TargetType: value.TargetType,
		TargetID: value.TargetID, TargetName: value.TargetName, TargetDetail: value.TargetDetail, Summary: value.Summary,
		IPAddress: value.IPAddress, UserAgent: value.UserAgent, CreatedAt: value.CreatedAt,
	}
}

type systemLogListAdminResponse struct {
	ID              uuid.UUID  `json:"id"`
	Environment     string     `json:"environment"`
	Service         string     `json:"service"`
	Version         string     `json:"version"`
	Level           string     `json:"level"`
	Category        string     `json:"category"`
	Event           string     `json:"event"`
	ErrorCode       string     `json:"error_code"`
	Message         string     `json:"message"`
	RequestID       string     `json:"request_id"`
	UserID          *uuid.UUID `json:"user_id,omitempty"`
	UserName        string     `json:"user_name,omitempty"`
	UserEmail       string     `json:"user_email,omitempty"`
	Method          string     `json:"method"`
	Path            string     `json:"path"`
	StatusCode      int        `json:"status_code"`
	ExternalService string     `json:"external_service,omitempty"`
	DurationMs      int64      `json:"duration_ms,omitempty"`
	IPAddress       string     `json:"ip_address,omitempty"`
	UserAgent       string     `json:"user_agent,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

func toSystemLogListAdminResponse(value usecase.SystemLogListResponse) systemLogListAdminResponse {
	return systemLogListAdminResponse{
		ID: value.ID, Environment: value.Environment, Service: value.Service, Version: value.Version,
		Level: value.Level, Category: value.Category, Event: value.Event, ErrorCode: value.ErrorCode,
		Message: value.Message, RequestID: value.RequestID, UserID: value.UserID, UserName: value.UserName,
		UserEmail: value.UserEmail, Method: value.Method, Path: value.Path, StatusCode: value.StatusCode,
		ExternalService: value.ExternalService, DurationMs: value.DurationMs, IPAddress: value.IPAddress,
		UserAgent: value.UserAgent, CreatedAt: value.CreatedAt,
	}
}
