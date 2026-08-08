package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"music-discovery/backend/entity"
)

func createAudit(tx *gorm.DB, a AuditContext) error {
	if a.ActorUserID != uuid.Nil && (a.ActorName == "" || a.ActorEmail == "" || a.ActorRole == "") {
		var actor entity.User
		if err := tx.Unscoped().Select("name", "email", "role").First(&actor, "id = ?", a.ActorUserID).Error; err == nil {
			a.ActorName, a.ActorEmail, a.ActorRole = actor.Name, actor.Email, actor.Role
		}
	}
	beforeValue, afterValue := minimalAuditDiff(a.Before, a.After)
	before, _ := json.Marshal(beforeValue)
	after, _ := json.Marshal(afterValue)
	log := entity.AuditLog{ID: uuid.New(), ActorUserID: a.ActorUserID, ActorRole: a.ActorRole, ActorName: a.ActorName, ActorEmail: a.ActorEmail, RequestID: a.RequestID, Action: a.Action, TargetType: a.TargetType, TargetID: a.TargetID, Reason: a.Reason, Summary: a.Summary, Before: before, After: after, IPAddress: a.IPAddress, UserAgent: a.UserAgent, CreatedAt: a.CreatedAt}
	if err := tx.Create(&log).Error; err != nil {
		return repoError(err)
	}
	return nil
}
func minimalAuditDiff(beforeValue, afterValue interface{}) (interface{}, interface{}) {
	switch {
	case beforeValue == nil, afterValue == nil:
		return beforeValue, afterValue
	}

	beforeMap, beforeOK := valueMap(beforeValue)
	afterMap, afterOK := valueMap(afterValue)
	switch {
	case !beforeOK, !afterOK:
		return beforeValue, afterValue
	}

	minimalBefore := make(map[string]interface{})
	minimalAfter := make(map[string]interface{})
	for key, after := range afterMap {
		before, exists := beforeMap[key]
		switch {
		case !exists, !valuesEqual(before, after):
			minimalBefore[key] = before
			minimalAfter[key] = after
		}
	}
	for key, before := range beforeMap {
		_, exists := afterMap[key]
		switch exists {
		case false:
			minimalBefore[key] = before
			minimalAfter[key] = nil
		}
	}
	return minimalBefore, minimalAfter
}

func valueMap(value interface{}) (map[string]interface{}, bool) {
	encoded, err := json.Marshal(value)
	switch {
	case err != nil:
		return nil, false
	}

	result := make(map[string]interface{})
	err = json.Unmarshal(encoded, &result)
	return result, err == nil
}

func valuesEqual(left, right interface{}) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	switch {
	case leftErr != nil, rightErr != nil:
		return false
	default:
		return bytes.Equal(leftJSON, rightJSON)
	}
}

func applyStringFilter(db *gorm.DB, column, value string) *gorm.DB {
	switch value {
	case "":
		return db
	default:
		return db.Where(column+" = ?", value)
	}
}

func applyLikeFilter(db *gorm.DB, column, value string) *gorm.DB {
	switch value {
	case "":
		return db
	default:
		return db.Where(column+" ILIKE ?", "%"+value+"%")
	}
}

func applyUUIDFilter(db *gorm.DB, column string, value *uuid.UUID) *gorm.DB {
	switch value {
	case nil:
		return db
	default:
		return db.Where(column+" = ?", *value)
	}
}

func applyIntFilter(db *gorm.DB, column string, value *int) *gorm.DB {
	switch value {
	case nil:
		return db
	default:
		return db.Where(column+" = ?", *value)
	}
}

func applyTimeFilter(db *gorm.DB, clause string, value *time.Time) *gorm.DB {
	switch value {
	case nil:
		return db
	default:
		return db.Where(clause, *value)
	}
}

func (r *adminRepository) ListAuditLogs(ctx context.Context, q PageQuery) ([]entity.AuditLog, int64, error) {
	db := addKeyword(r.db.WithContext(ctx).Model(&entity.AuditLog{}), q.Keyword, "action", "target_type", "target_id", "summary", "reason", "request_id", "actor_name", "actor_email")
	db = applyStringFilter(db, "action", q.Action)
	db = applyUUIDFilter(db, "actor_user_id", q.ActorID)
	db = applyStringFilter(db, "target_type", q.TargetType)
	db = applyStringFilter(db, "target_id", q.TargetID)
	db = applyStringFilter(db, "request_id", q.RequestID)
	db = applyTimeFilter(db, "created_at >= ?", q.From)
	db = applyTimeFilter(db, "created_at < ?", q.To)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, repoError(err)
	}

	var items []entity.AuditLog
	db = db.Omit("before", "after", "reason")
	if err := db.Order(safeOrder(q.Sort, q.Order, map[string]string{"action": "action", "created_at": "created_at"}, "created_at")).Offset(offset(q)).Limit(q.Limit).Find(&items).Error; err != nil {
		return nil, 0, repoError(err)
	}
	return items, total, nil
}

func (r *adminRepository) FindAuditLog(ctx context.Context, id uuid.UUID) (*entity.AuditLog, error) {
	var value entity.AuditLog
	if err := r.db.WithContext(ctx).First(&value, "id = ?", id).Error; err != nil {
		return nil, repoError(err)
	}
	return &value, nil
}
func (r *adminRepository) ListSystemLogs(ctx context.Context, q PageQuery) ([]entity.SystemLog, int64, error) {
	db := addKeyword(r.db.WithContext(ctx).Model(&entity.SystemLog{}), q.Keyword, "level", "category", "event", "error_code", "message", "request_id", "path", "external_service", "error_detail")
	db = applyStringFilter(db, "level", q.Level)
	db = applyStringFilter(db, "category", q.Category)
	db = applyStringFilter(db, "event", q.Event)
	db = applyUUIDFilter(db, "user_id", q.UserID)
	db = applyStringFilter(db, "request_id", q.RequestID)
	db = applyIntFilter(db, "status_code", q.StatusCode)
	db = applyStringFilter(db, "external_service", q.ExternalService)
	db = applyStringFilter(db, "method", q.Method)
	db = applyLikeFilter(db, "path", q.Path)
	db = applyTimeFilter(db, "created_at >= ?", q.From)
	db = applyTimeFilter(db, "created_at < ?", q.To)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, repoError(err)
	}

	var items []entity.SystemLog
	db = db.Omit("metadata", "error_detail", "stack_trace")
	if err := db.Order(safeOrder(q.Sort, q.Order, map[string]string{"level": "level", "category": "category", "event": "event", "status_code": "status_code", "created_at": "created_at"}, "created_at")).Offset(offset(q)).Limit(q.Limit).Find(&items).Error; err != nil {
		return nil, 0, repoError(err)
	}
	return items, total, nil
}

func (r *adminRepository) FindSystemLog(ctx context.Context, id uuid.UUID) (*entity.SystemLog, error) {
	var value entity.SystemLog
	if err := r.db.WithContext(ctx).First(&value, "id = ?", id).Error; err != nil {
		return nil, repoError(err)
	}
	return &value, nil
}
func (r *adminRepository) CreateSystemLog(ctx context.Context, value *entity.SystemLog) error {
	if err := r.db.WithContext(ctx).Create(value).Error; err != nil {
		return repoError(err)
	}
	return nil
}

func (r *adminRepository) ResolveAuditTarget(ctx context.Context, targetType, targetID string) (string, string) {
	id, err := uuid.Parse(targetID)
	if err != nil {
		return targetID, ""
	}
	switch targetType {
	case "user", "role":
		var value entity.User
		if err := r.db.WithContext(ctx).Unscoped().Select("name", "email").First(&value, "id = ?", id).Error; err == nil {
			return value.Name, value.Email
		}
	case "artist":
		var value entity.Artist
		if err := r.db.WithContext(ctx).Unscoped().Select("name", "spotify_artist_id").First(&value, "id = ?", id).Error; err == nil {
			return value.Name, value.SpotifyArtistID
		}
	case "song":
		var value entity.Song
		if err := r.db.WithContext(ctx).Unscoped().Select("name", "spotify_track_id").First(&value, "id = ?", id).Error; err == nil {
			return value.Name, value.SpotifyTrackID
		}
	case "tag":
		var value entity.Tag
		if err := r.db.WithContext(ctx).Unscoped().Select("name", "slug").First(&value, "id = ?", id).Error; err == nil {
			return value.Name, value.Slug
		}
	}
	return targetID, ""
}
