package repository

import (
	"context"
	"errors"
	"time"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IFeedbackRepository interface {
	FindTarget(ctx context.Context, recommendationDetailID uuid.UUID) (*entity.FeedbackTarget, error)
	FindTargetForUpdate(ctx context.Context, tx ITransaction, recommendationDetailID uuid.UUID) (*entity.FeedbackTarget, error)
	FindLearningProfileForUpdate(ctx context.Context, tx ITransaction, userID uuid.UUID) (*entity.UserProfile, error)
	FindByDetailID(ctx context.Context, tx ITransaction, recommendationDetailID uuid.UUID) (*entity.Feedback, error)
	Create(ctx context.Context, tx ITransaction, feedback *entity.Feedback) error
	Update(ctx context.Context, tx ITransaction, feedback *entity.Feedback) error
	ListLearningSources(ctx context.Context, tx ITransaction, userID uuid.UUID) ([]entity.FeedbackLearningSource, error)
	ReplaceLearningSnapshot(ctx context.Context, tx ITransaction, userID, profileID uuid.UUID, snapshot entity.FeedbackLearningSnapshot, updatedAt time.Time) error
}

type feedbackRepository struct{ db *gorm.DB }

func NewFeedbackRepository(db *gorm.DB) IFeedbackRepository { return &feedbackRepository{db: db} }

func (r *feedbackRepository) FindTarget(ctx context.Context, detailID uuid.UUID) (*entity.FeedbackTarget, error) {
	return r.findTarget(r.db.WithContext(ctx), detailID, false)
}

func (r *feedbackRepository) FindTargetForUpdate(ctx context.Context, tx ITransaction, detailID uuid.UUID) (*entity.FeedbackTarget, error) {
	db, err := txDB(ctx, tx)
	if err != nil {
		return nil, err
	}
	return r.findTarget(db, detailID, true)
}

func (r *feedbackRepository) findTarget(db *gorm.DB, detailID uuid.UUID, lock bool) (*entity.FeedbackTarget, error) {
	var detail entity.RecommendationDetail
	query := db.Preload("Song.Artist")
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := query.First(&detail, "id = ?", detailID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, entity.ErrRecommendationDetailNotFound
		}
		return nil, entity.ErrInternal
	}
	var recommendation entity.Recommendation
	if err := db.First(&recommendation, "id = ?", detail.RecommendationID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, entity.ErrRecommendationNotFound
		}
		return nil, entity.ErrInternal
	}
	if detail.Song.ID == uuid.Nil {
		return nil, entity.ErrSongNotFound
	}
	if detail.Song.Artist.ID == uuid.Nil {
		return nil, entity.ErrArtistNotFound
	}
	var tags []entity.Tag
	if err := db.Raw(`
		SELECT DISTINCT t.* FROM tags t
		LEFT JOIN song_tags st ON st.tag_id = t.id AND st.song_id = ?
		LEFT JOIN artist_tags at ON at.tag_id = t.id AND at.artist_id = ?
		WHERE t.deleted_at IS NULL AND (st.id IS NOT NULL OR at.id IS NOT NULL)
	`, detail.SongID, detail.Song.ArtistID).Scan(&tags).Error; err != nil {
		return nil, entity.ErrInternal
	}
	return &entity.FeedbackTarget{Detail: detail, Recommendation: recommendation, Song: detail.Song, Artist: detail.Song.Artist, Tags: tags}, nil
}

func (r *feedbackRepository) FindLearningProfileForUpdate(ctx context.Context, tx ITransaction, userID uuid.UUID) (*entity.UserProfile, error) {
	db, err := txDB(ctx, tx)
	if err != nil {
		return nil, err
	}
	var profile entity.UserProfile
	err = db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("GenreWeights").Preload("ElementWeights").Preload("AtmosphereWeights").
		First(&profile, "user_id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, entity.ErrUserProfileRequired
	}
	if err != nil {
		return nil, entity.ErrInternal
	}
	return &profile, nil
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

func (r *feedbackRepository) ListLearningSources(ctx context.Context, tx ITransaction, userID uuid.UUID) ([]entity.FeedbackLearningSource, error) {
	db, err := txDB(ctx, tx)
	if err != nil {
		return nil, err
	}
	type row struct {
		FeedbackID   uuid.UUID
		FeedbackType entity.FeedbackType
		ArtistID     uuid.UUID
		SongID       uuid.UUID
		Exploration  bool
		ChoiceID     *uuid.UUID
		Category     *string
	}
	var rows []row
	if err := db.Raw(`
		SELECT f.id AS feedback_id, f.feedback_type, f.artist_id, f.song_id,
		       (rd.recommendation_type = 'TRY_SOMETHING_NEW' OR r.source = 'DAILY_DISCOVERY') AS exploration,
		       qc.id AS choice_id, LOWER(t.category) AS category
		FROM feedbacks f
		JOIN recommendation_details rd ON rd.id = f.recommendation_detail_id
		JOIN recommendations r ON r.id = rd.recommendation_id
		LEFT JOIN tags t ON t.deleted_at IS NULL AND (
			EXISTS (SELECT 1 FROM song_tags st WHERE st.song_id = f.song_id AND st.tag_id = t.id) OR
			EXISTS (SELECT 1 FROM artist_tags at WHERE at.artist_id = f.artist_id AND at.tag_id = t.id)
		)
		LEFT JOIN question_choices qc
		  ON LOWER(REPLACE(qc.code, '_', '-')) = LOWER(t.slug)
		 AND qc.is_active = TRUE
		WHERE f.user_id = ?
		ORDER BY f.id
	`, userID).Scan(&rows).Error; err != nil {
		return nil, entity.ErrInternal
	}
	byID := make(map[uuid.UUID]*entity.FeedbackLearningSource)
	order := make([]uuid.UUID, 0)
	for _, value := range rows {
		source, exists := byID[value.FeedbackID]
		if !exists {
			source = &entity.FeedbackLearningSource{FeedbackID: value.FeedbackID, FeedbackType: value.FeedbackType, ArtistID: value.ArtistID, SongID: value.SongID, Exploration: value.Exploration}
			byID[value.FeedbackID] = source
			order = append(order, value.FeedbackID)
		}
		if value.ChoiceID != nil && value.Category != nil {
			source.Tags = append(source.Tags, entity.FeedbackLearningTag{ChoiceID: *value.ChoiceID, Category: *value.Category})
		}
	}
	result := make([]entity.FeedbackLearningSource, 0, len(order))
	for _, id := range order {
		result = append(result, *byID[id])
	}
	return result, nil
}

func (r *feedbackRepository) ReplaceLearningSnapshot(ctx context.Context, tx ITransaction, userID, profileID uuid.UUID, snapshot entity.FeedbackLearningSnapshot, updatedAt time.Time) error {
	db, err := txDB(ctx, tx)
	if err != nil {
		return err
	}
	if err := db.Where("user_id = ?", userID).Delete(&entity.UserArtistWeight{}).Error; err != nil {
		return entity.ErrInternal
	}
	if len(snapshot.ArtistWeights) > 0 {
		if err := db.Create(&snapshot.ArtistWeights).Error; err != nil {
			return entity.ErrInternal
		}
	}
	if err := db.Where("user_id = ?", userID).Delete(&entity.UserSongWeight{}).Error; err != nil {
		return entity.ErrInternal
	}
	if len(snapshot.SongWeights) > 0 {
		if err := db.Create(&snapshot.SongWeights).Error; err != nil {
			return entity.ErrInternal
		}
	}
	if err := replaceProfileWeights(db, profileID, snapshot.GenreWeights, &entity.UserGenreWeight{}); err != nil {
		return err
	}
	if err := replaceProfileWeights(db, profileID, snapshot.ElementWeights, &entity.UserElementWeight{}); err != nil {
		return err
	}
	if err := replaceProfileWeights(db, profileID, snapshot.AtmosphereWeights, &entity.UserAtmosphereWeight{}); err != nil {
		return err
	}
	result := db.Model(&entity.UserProfile{}).Where("id = ?", profileID).Updates(map[string]interface{}{
		"discovery_adjustment": snapshot.DiscoveryAdjustment,
		"updated_at":           updatedAt,
	})
	if result.Error != nil {
		return entity.ErrInternal
	}
	if result.RowsAffected == 0 {
		return entity.ErrUserProfileRequired
	}
	return nil
}

func replaceProfileWeights[T entity.UserGenreWeight | entity.UserElementWeight | entity.UserAtmosphereWeight](db *gorm.DB, profileID uuid.UUID, values []T, model interface{}) error {
	if err := db.Where("profile_id = ?", profileID).Delete(model).Error; err != nil {
		return entity.ErrInternal
	}
	if len(values) > 0 {
		if err := db.Create(&values).Error; err != nil {
			return entity.ErrInternal
		}
	}
	return nil
}
