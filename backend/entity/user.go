package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRole = Role

const (
	UserRoleUser         = RoleUser
	UserRoleContentAdmin = RoleContentAdmin
	UserRoleSystemAdmin  = RoleSystemAdmin
)

type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
)

type User struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name         string         `json:"name" gorm:"size:50;not null"`
	Email        string         `json:"email" gorm:"size:254;not null;uniqueIndex"`
	PasswordHash string         `json:"-" gorm:"not null"`
	Role         UserRole       `json:"role" gorm:"size:32;not null"`
	Status       UserStatus     `json:"status" gorm:"size:32;not null"`
	TokenVersion uint64         `json:"-" gorm:"not null;default:0"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}
