package entity

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID       uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	TokenHash    string     `json:"-" gorm:"size:64;not null;uniqueIndex"`
	FamilyID     uuid.UUID  `json:"family_id" gorm:"type:uuid;not null;index"`
	TokenVersion uint64     `json:"-" gorm:"not null"`
	ReplacedByID *uuid.UUID `json:"replaced_by_id,omitempty" gorm:"type:uuid"`
	ExpiresAt    time.Time  `json:"expires_at" gorm:"not null;index"`
	UsedAt       *time.Time `json:"used_at,omitempty"` // legacy column; rotation uses RevokedAt
	RevokedAt    *time.Time `json:"revoked_at,omitempty" gorm:"index"`
	CreatedAt    time.Time  `json:"created_at" gorm:"not null"`
}

func (t *RefreshToken) IsExpired(now time.Time) bool { return !now.Before(t.ExpiresAt) }
func (t *RefreshToken) IsUsed() bool                 { return t.UsedAt != nil }
func (t *RefreshToken) IsRevoked() bool              { return t.RevokedAt != nil }
