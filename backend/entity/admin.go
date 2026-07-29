package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Artist struct {
	ID              uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SpotifyArtistID string         `json:"spotify_artist_id" gorm:"size:64;not null;uniqueIndex"`
	Name            string         `json:"name" gorm:"size:200;not null;index"`
	ImageURL        string         `json:"image_url"`
	SpotifyURL      string         `json:"spotify_url" gorm:"not null"`
	Popularity      int            `json:"popularity" gorm:"not null"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

type Song struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ArtistID       uuid.UUID      `json:"artist_id" gorm:"type:uuid;not null;index"`
	Artist         Artist         `json:"artist,omitempty" gorm:"foreignKey:ArtistID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;-:migration"`
	SpotifyTrackID string         `json:"spotify_track_id" gorm:"size:64;not null;uniqueIndex"`
	Name           string         `json:"name" gorm:"size:200;not null;index"`
	AlbumName      string         `json:"album_name" gorm:"size:200;not null"`
	ImageURL       string         `json:"image_url"`
	SpotifyURL     string         `json:"spotify_url" gorm:"not null"`
	PreviewURL     string         `json:"preview_url"`
	DurationMs     int            `json:"duration_ms" gorm:"not null"`
	Popularity     int            `json:"popularity" gorm:"not null"`
	ReleaseDate    time.Time      `json:"release_date"`
	Explicit       bool           `json:"explicit"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

type Tag struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string         `json:"name" gorm:"size:100;not null;uniqueIndex"`
	Slug        string         `json:"slug" gorm:"size:100;not null;uniqueIndex"`
	Category    string         `json:"category" gorm:"size:50;not null;index"`
	Description string         `json:"description" gorm:"size:500"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

type ArtistTag struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ArtistID  uuid.UUID `json:"artist_id" gorm:"type:uuid;not null;uniqueIndex:idx_artist_tag"`
	TagID     uuid.UUID `json:"tag_id" gorm:"type:uuid;not null;uniqueIndex:idx_artist_tag"`
	CreatedAt time.Time `json:"created_at"`
}

type SongTag struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SongID    uuid.UUID `json:"song_id" gorm:"type:uuid;not null;uniqueIndex:idx_song_tag"`
	TagID     uuid.UUID `json:"tag_id" gorm:"type:uuid;not null;uniqueIndex:idx_song_tag"`
	Weight    float64   `json:"weight" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AuditLog struct {
	ID          uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ActorUserID uuid.UUID       `json:"actor_user_id" gorm:"type:uuid;not null;index"`
	ActorRole   Role            `json:"actor_role" gorm:"size:32;index"`
	ActorName   string          `json:"actor_name" gorm:"size:100"`
	ActorEmail  string          `json:"actor_email" gorm:"size:254"`
	RequestID   string          `json:"request_id" gorm:"size:100;index"`
	Action      string          `json:"action" gorm:"size:64;not null;index"`
	TargetType  string          `json:"target_type" gorm:"size:64;not null;index"`
	TargetID    string          `json:"target_id" gorm:"size:128;not null"`
	Reason      string          `json:"reason" gorm:"size:500"`
	Before      json.RawMessage `json:"before,omitempty" gorm:"type:jsonb"`
	After       json.RawMessage `json:"after,omitempty" gorm:"type:jsonb"`
	Summary     string          `json:"summary" gorm:"size:500"`
	IPAddress   string          `json:"ip_address" gorm:"size:64"`
	UserAgent   string          `json:"user_agent" gorm:"size:500"`
	CreatedAt   time.Time       `json:"created_at" gorm:"index"`
}

type SystemLog struct {
	ID              uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Level           string          `json:"level" gorm:"size:16;not null;index"`
	Category        string          `json:"category" gorm:"size:32;not null;index"`
	Event           string          `json:"event" gorm:"size:64;not null;index"`
	ErrorCode       string          `json:"error_code" gorm:"size:64;index"`
	Message         string          `json:"message" gorm:"size:1000;not null"`
	RequestID       string          `json:"request_id" gorm:"size:100;index"`
	UserID          *uuid.UUID      `json:"user_id,omitempty" gorm:"type:uuid;index"`
	Method          string          `json:"method" gorm:"size:16"`
	Path            string          `json:"path" gorm:"size:500"`
	StatusCode      int             `json:"status_code" gorm:"index"`
	Metadata        json.RawMessage `json:"metadata,omitempty" gorm:"type:jsonb"`
	StackTrace      string          `json:"stack_trace,omitempty" gorm:"type:text"`
	ExternalService string          `json:"external_service,omitempty" gorm:"size:100;index"`
	DurationMs      int64           `json:"duration_ms,omitempty"`
	ErrorDetail     string          `json:"error_detail,omitempty" gorm:"type:text"`
	IPAddress       string          `json:"ip_address,omitempty" gorm:"size:64"`
	UserAgent       string          `json:"user_agent,omitempty" gorm:"size:500"`
	CreatedAt       time.Time       `json:"created_at" gorm:"index"`
}
