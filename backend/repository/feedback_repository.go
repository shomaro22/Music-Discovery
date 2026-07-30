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

type FeedbackTarget struct {
	Detail         entity.RecommendationDetail
	Recommendation entity.Recommendation
	Song           entity.Song
	Artist         entity.Artist
	Tags           []entity.Tag
}

type IFeedbackRepository interface {
	FindTarget(ctx context.Context, recommendationDetailID uuid.UUID) (*FeedbackTarget, error)
	FindByDetailID(ctx context.Context, tx ITransaction, recommendationDetailID uuid.UUID) (*entity.Feedback, error)
	Create(ctx context.Context, tx ITransaction, feedback *entity.Feedback) error
	Update(ctx context.Context, tx ITransaction, feedback *entity.Feedback) error
	ApplyArtistWeight(ctx context.Context, tx ITransaction, userID, artistID uuid.UUID, delta, minimum, maximum float64) error
	ApplySongWeight(ctx context.Context, tx ITransaction, userID, songID uuid.UUID, delta, minimum, maximum float64) error
	ApplyProfileTagWeight(ctx context.Context, tx ITransaction, userID uuid.UUID, tag entity.Tag, delta, minimum, maximum float64) error
	ApplyDiscoveryAdjustment(ctx context.Context, tx ITransaction, userID uuid.UUID, delta, minimum, maximum float64) error
}

type feedbackRepository struct{ db *gorm.DB }

func NewFeedbackRepository(db *gorm.DB) IFeedbackRepository { return &feedbackRepository{db: db} }

func (r *feedbackRepository) FindTarget(ctx context.Context, detailID uuid.UUID) (*FeedbackTarget, error) {
	var detail entity.RecommendationDetail
	if err := r.db.WithContext(ctx).Preload("Song.Artist").First(&detail, "id = ?", detailID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, entity.ErrRecommendationDetailNotFound
		}
		return nil, entity.ErrInternal
	}
	var recommendation entity.Recommendation
	if err := r.db.WithContext(ctx).First(&recommendation, "id = ?", detail.RecommendationID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, entity.ErrRecommendationNotFound
		}
		return nil, entity.ErrInternal
	}
	tags := make([]entity.Tag, 0)
	if err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT t.* FROM tags t
		LEFT JOIN song_tags st ON st.tag_id = t.id AND st.song_id = ?
		LEFT JOIN artist_tags at ON at.tag_id = t.id AND at.artist_id = ?
		WHERE t.deleted_at IS NULL AND (st.id IS NOT NULL OR at.id IS NOT NULL)
	`, detail.SongID, detail.Song.ArtistID).Scan(&tags).Error; err != nil {
		return nil, entity.ErrInternal
	}
	return &FeedbackTarget{Detail: detail, Recommendation: recommendation, Song: detail.Song, Artist: detail.Song.Artist, Tags: tags}, nil
}

func txDB(ctx context.Context, tx ITransaction) (*gorm.DB, error) {
	db, ok := transactionDB(tx)
	if !ok {
		return nil, entity.ErrInvalidInput
	}
	return db.WithContext(ctx), nil
}

func (r *feedbackRepository) FindByDetailID(ctx context.Context, tx ITransaction, detailID uuid.UUID) (*entity.Feedback, error) {
	db, err := txDB(ctx, tx)
	if err != nil {
		return nil, err
	}
	var value entity.Feedback
	result := db.First(&value, "recommendation_detail_id = ?", detailID)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, entity.ErrNotFound
	}
	if result.Error != nil {
		return nil, entity.ErrInternal
	}
	return &value, nil
}
func (r *feedbackRepository) Create(ctx context.Context, tx ITransaction, value *entity.Feedback) error {
	db, err := txDB(ctx, tx)
	if err != nil || value == nil {
		return entity.ErrInvalidInput
	}
	if err := db.Create(value).Error; err != nil {
		return entity.ErrFeedbackSaveFailed
	}
	return nil
}
func (r *feedbackRepository) Update(ctx context.Context, tx ITransaction, value *entity.Feedback) error {
	db, err := txDB(ctx, tx)
	if err != nil || value == nil {
		return entity.ErrInvalidInput
	}
	if err := db.Model(&entity.Feedback{}).Where("id = ?", value.ID).Updates(struct {
		Type      entity.FeedbackType `gorm:"column:feedback_type"`
		UpdatedAt time.Time
	}{Type: value.Type, UpdatedAt: value.UpdatedAt}).Error; err != nil {
		return entity.ErrFeedbackUpdateFailed
	}
	return nil
}
func (r *feedbackRepository) ApplyArtistWeight(ctx context.Context, tx ITransaction, userID, artistID uuid.UUID, delta, minimum, maximum float64) error {
	db, err := txDB(ctx, tx)
	if err != nil {
		return err
	}
	return applySimpleWeight(db, "user_artist_weights", "artist_id", userID, artistID, delta, minimum, maximum)
}
func (r *feedbackRepository) ApplySongWeight(ctx context.Context, tx ITransaction, userID, songID uuid.UUID, delta, minimum, maximum float64) error {
	db, err := txDB(ctx, tx)
	if err != nil {
		return err
	}
	return applySimpleWeight(db, "user_song_weights", "song_id", userID, songID, delta, minimum, maximum)
}
func applySimpleWeight(db *gorm.DB, table, targetColumn string, userID, targetID uuid.UUID, delta, minimum, maximum float64) error {
	query := "INSERT INTO " + table + " (id,user_id," + targetColumn + ",weight,created_at,updated_at) VALUES (gen_random_uuid(),?,?,LEAST(?,GREATEST(?,?)),NOW(),NOW()) ON CONFLICT (user_id," + targetColumn + ") DO UPDATE SET weight=LEAST(?,GREATEST(?," + table + ".weight + ?)),updated_at=NOW()"
	if err := db.Exec(query, userID, targetID, maximum, minimum, delta, maximum, minimum, delta).Error; err != nil {
		return entity.ErrInternal
	}
	return nil
}
func (r *feedbackRepository) ApplyProfileTagWeight(ctx context.Context, tx ITransaction, userID uuid.UUID, tag entity.Tag, delta, minimum, maximum float64) error {
	db, err := txDB(ctx, tx)
	if err != nil {
		return err
	}
	category := strings.ToLower(strings.TrimSpace(tag.Category))
	choiceCode := strings.ToLower(strings.TrimSpace(tag.Slug))
	choiceCode = strings.TrimPrefix(choiceCode, category+"-")
	table := ""
	switch category {
	case "genre":
		table = "user_genre_weights"
	case "element":
		table = "user_element_weights"
	case "atmosphere":
		table = "user_atmosphere_weights"
	default:
		return nil
	}
	var row struct{ ProfileID, ChoiceID uuid.UUID }
	if err := db.Raw(`SELECT up.id profile_id, qc.id choice_id FROM user_profiles up JOIN question_choices qc ON LOWER(qc.code)=LOWER(?) WHERE up.user_id=? AND qc.is_active = TRUE LIMIT 1`, tag.Slug, userID).Scan(&row).Error; err != nil {
		return entity.ErrInternal
	}
	if row.ProfileID == uuid.Nil || row.ChoiceID == uuid.Nil {
		return nil
	}
	query := "INSERT INTO " + table + " (id,profile_id,choice_id,question_weight,feedback_weight,weight,created_at,updated_at) VALUES (gen_random_uuid(),?,?,0,LEAST(?,GREATEST(?,?)),LEAST(?,GREATEST(?,?)),NOW(),NOW()) ON CONFLICT (profile_id,choice_id) DO UPDATE SET feedback_weight=LEAST(?,GREATEST(?," + table + ".feedback_weight + ?)),weight=" + table + ".question_weight + LEAST(?,GREATEST(?," + table + ".feedback_weight + ?)),updated_at=NOW()"
	if err := db.Exec(query, row.ProfileID, row.ChoiceID, maximum, minimum, delta, maximum, minimum, delta, maximum, minimum, delta, maximum, minimum, delta).Error; err != nil {
		return entity.ErrInternal
	}
	return nil
}
func (r *feedbackRepository) ApplyDiscoveryAdjustment(ctx context.Context, tx ITransaction, userID uuid.UUID, delta, minimum, maximum float64) error {
	db, err := txDB(ctx, tx)
	if err != nil {
		return err
	}
	result := db.Exec(`UPDATE user_profiles SET discovery_adjustment=LEAST(?,GREATEST(?,discovery_adjustment+?)),updated_at=NOW() WHERE user_id=?`, maximum, minimum, delta, userID)
	if result.Error != nil {
		return entity.ErrInternal
	}
	if result.RowsAffected == 0 {
		return entity.ErrUserProfileRequired
	}
	return nil
}
