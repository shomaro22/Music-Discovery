package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	FindTargetForUpdate(ctx context.Context, tx ITransaction, recommendationDetailID uuid.UUID) (*FeedbackTarget, error)
	LockUserLearningState(ctx context.Context, tx ITransaction, userID uuid.UUID) error
	FindByDetailID(ctx context.Context, tx ITransaction, recommendationDetailID uuid.UUID) (*entity.Feedback, error)
	Create(ctx context.Context, tx ITransaction, feedback *entity.Feedback) error
	Update(ctx context.Context, tx ITransaction, feedback *entity.Feedback) error
	ApplyArtistWeight(ctx context.Context, tx ITransaction, userID, artistID uuid.UUID, delta, minimum, maximum float64) error
	ApplySongWeight(ctx context.Context, tx ITransaction, userID, songID uuid.UUID, delta, minimum, maximum float64) error
	ApplyProfileTagWeight(ctx context.Context, tx ITransaction, userID uuid.UUID, tag entity.Tag, delta, minimum, maximum float64) error
	ApplyDiscoveryAdjustment(ctx context.Context, tx ITransaction, userID uuid.UUID, delta, minimum, maximum float64) error
	RecalculateLearningWeights(ctx context.Context, tx ITransaction, userID uuid.UUID, deltas map[entity.FeedbackType]FeedbackWeightDelta, minimum, maximum, minimumDiscovery, maximumDiscovery float64) error
}

type FeedbackWeightDelta struct {
	Genre, Artist, Song, Profile, Discovery float64
}

type feedbackRepository struct{ db *gorm.DB }

func NewFeedbackRepository(db *gorm.DB) IFeedbackRepository { return &feedbackRepository{db: db} }

func (r *feedbackRepository) FindTarget(ctx context.Context, detailID uuid.UUID) (*FeedbackTarget, error) {
	return r.findTarget(r.db.WithContext(ctx), detailID, false)
}

func (r *feedbackRepository) FindTargetForUpdate(ctx context.Context, tx ITransaction, detailID uuid.UUID) (*FeedbackTarget, error) {
	db, err := txDB(ctx, tx)
	if err != nil {
		return nil, err
	}
	return r.findTarget(db, detailID, true)
}

func (r *feedbackRepository) findTarget(db *gorm.DB, detailID uuid.UUID, lock bool) (*FeedbackTarget, error) {
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
	tags := make([]entity.Tag, 0)
	if err := db.Raw(`
		SELECT DISTINCT t.* FROM tags t
		LEFT JOIN song_tags st ON st.tag_id = t.id AND st.song_id = ?
		LEFT JOIN artist_tags at ON at.tag_id = t.id AND at.artist_id = ?
		WHERE t.deleted_at IS NULL AND (st.id IS NOT NULL OR at.id IS NOT NULL)
	`, detail.SongID, detail.Song.ArtistID).Scan(&tags).Error; err != nil {
		return nil, entity.ErrInternal
	}
	return &FeedbackTarget{Detail: detail, Recommendation: recommendation, Song: detail.Song, Artist: detail.Song.Artist, Tags: tags}, nil
}

func (r *feedbackRepository) LockUserLearningState(ctx context.Context, tx ITransaction, userID uuid.UUID) error {
	db, err := txDB(ctx, tx)
	if err != nil {
		return err
	}
	var profile entity.UserProfile
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&profile, "user_id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.ErrUserProfileRequired
		}
		return entity.ErrInternal
	}
	return nil
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
	case "element", "feature":
		table = "user_element_weights"
	case "atmosphere", "scene":
		table = "user_atmosphere_weights"
	default:
		return nil
	}
	var row struct{ ProfileID, ChoiceID uuid.UUID }
	if err := db.Raw(`SELECT up.id profile_id, qc.id choice_id FROM user_profiles up JOIN question_choices qc ON LOWER(REPLACE(qc.code, '_', '-'))=LOWER(?) WHERE up.user_id=? AND qc.is_active = TRUE LIMIT 1`, tag.Slug, userID).Scan(&row).Error; err != nil {
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

func (r *feedbackRepository) RecalculateLearningWeights(ctx context.Context, tx ITransaction, userID uuid.UUID, deltas map[entity.FeedbackType]FeedbackWeightDelta, minimum, maximum, minimumDiscovery, maximumDiscovery float64) error {
	db, err := txDB(ctx, tx)
	if err != nil {
		return err
	}
	value := func(t entity.FeedbackType, pick func(FeedbackWeightDelta) float64) float64 { return pick(deltas[t]) }
	likeA, maybeA, skipA := value(entity.FeedbackTypeLike, func(v FeedbackWeightDelta) float64 { return v.Artist }), value(entity.FeedbackTypeMaybe, func(v FeedbackWeightDelta) float64 { return v.Artist }), value(entity.FeedbackTypeSkip, func(v FeedbackWeightDelta) float64 { return v.Artist })
	likeS, maybeS, skipS := value(entity.FeedbackTypeLike, func(v FeedbackWeightDelta) float64 { return v.Song }), value(entity.FeedbackTypeMaybe, func(v FeedbackWeightDelta) float64 { return v.Song }), value(entity.FeedbackTypeSkip, func(v FeedbackWeightDelta) float64 { return v.Song })
	if err := replaceSimpleWeights(db, "user_artist_weights", "artist_id", userID, likeA, maybeA, skipA, minimum, maximum); err != nil {
		return err
	}
	if err := replaceSimpleWeights(db, "user_song_weights", "song_id", userID, likeS, maybeS, skipS, minimum, maximum); err != nil {
		return err
	}
	if err := r.recalculateProfileWeights(db, userID, deltas, minimum, maximum); err != nil {
		return err
	}
	likeD, maybeD, skipD := value(entity.FeedbackTypeLike, func(v FeedbackWeightDelta) float64 { return v.Discovery }), value(entity.FeedbackTypeMaybe, func(v FeedbackWeightDelta) float64 { return v.Discovery }), value(entity.FeedbackTypeSkip, func(v FeedbackWeightDelta) float64 { return v.Discovery })
	return recalculateDiscovery(db, userID, likeD, maybeD, skipD, minimumDiscovery, maximumDiscovery)
}

func replaceSimpleWeights(db *gorm.DB, table, targetColumn string, userID uuid.UUID, like, maybe, skip, minimum, maximum float64) error {
	if err := db.Exec("DELETE FROM "+table+" WHERE user_id = ?", userID).Error; err != nil {
		return entity.ErrInternal
	}
	query := "INSERT INTO " + table + " (id,user_id," + targetColumn + ",weight,created_at,updated_at) SELECT gen_random_uuid(),user_id," + targetColumn + ",LEAST(?,GREATEST(?,SUM(CASE feedback_type WHEN 'LIKE' THEN ? WHEN 'MAYBE' THEN ? WHEN 'SKIP' THEN ? ELSE 0 END))),NOW(),NOW() FROM feedbacks WHERE user_id=? GROUP BY user_id," + targetColumn
	if err := db.Exec(query, maximum, minimum, like, maybe, skip, userID).Error; err != nil {
		return entity.ErrInternal
	}
	return nil
}

func (r *feedbackRepository) recalculateProfileWeights(db *gorm.DB, userID uuid.UUID, deltas map[entity.FeedbackType]FeedbackWeightDelta, minimum, maximum float64) error {
	var profileID uuid.UUID
	if err := db.Model(&entity.UserProfile{}).Select("id").Where("user_id = ?", userID).Scan(&profileID).Error; err != nil || profileID == uuid.Nil {
		return entity.ErrUserProfileRequired
	}
	for _, table := range []string{"user_genre_weights", "user_element_weights", "user_atmosphere_weights"} {
		if err := db.Exec("UPDATE "+table+" SET feedback_weight=0, weight=LEAST(?,GREATEST(?,question_weight)), updated_at=NOW() WHERE profile_id=?", maximum, minimum, profileID).Error; err != nil {
			return entity.ErrInternal
		}
	}
	type tagWeight struct {
		ChoiceID uuid.UUID
		Category string
		Weight   float64
	}
	rows := []tagWeight{}
	query := `WITH feedback_tags AS (
		SELECT DISTINCT f.id feedback_id, f.feedback_type, t.id tag_id, LOWER(t.category) category, t.slug
		FROM feedbacks f
		JOIN songs s ON s.id=f.song_id
		JOIN recommendations r ON r.user_id=f.user_id
		JOIN recommendation_details rd ON rd.recommendation_id=r.id AND rd.id=f.recommendation_detail_id
		JOIN tags t ON t.deleted_at IS NULL
		LEFT JOIN song_tags st ON st.song_id=s.id AND st.tag_id=t.id
		LEFT JOIN artist_tags at ON at.artist_id=f.artist_id AND at.tag_id=t.id
		WHERE f.user_id=? AND (st.id IS NOT NULL OR at.id IS NOT NULL)
	)
	SELECT qc.id choice_id, ft.category,
	SUM(CASE ft.feedback_type WHEN 'LIKE' THEN CASE WHEN ft.category='genre' THEN ? ELSE ? END WHEN 'MAYBE' THEN CASE WHEN ft.category='genre' THEN ? ELSE ? END WHEN 'SKIP' THEN CASE WHEN ft.category='genre' THEN ? ELSE ? END ELSE 0 END) weight
	FROM feedback_tags ft JOIN question_choices qc ON LOWER(REPLACE(qc.code, '_', '-'))=LOWER(ft.slug) AND qc.is_active=TRUE
	WHERE ft.category IN ('genre','element','atmosphere','feature','scene') GROUP BY qc.id,ft.category`
	if err := db.Raw(query, userID, deltas[entity.FeedbackTypeLike].Genre, deltas[entity.FeedbackTypeLike].Profile, deltas[entity.FeedbackTypeMaybe].Genre, deltas[entity.FeedbackTypeMaybe].Profile, deltas[entity.FeedbackTypeSkip].Genre, deltas[entity.FeedbackTypeSkip].Profile).Scan(&rows).Error; err != nil {
		return entity.ErrInternal
	}
	for _, row := range rows {
		table := ""
		switch row.Category {
		case "genre":
			table = "user_genre_weights"
		case "element", "feature":
			table = "user_element_weights"
		case "atmosphere", "scene":
			table = "user_atmosphere_weights"
		}
		if table == "" {
			continue
		}
		q := "INSERT INTO " + table + " (id,profile_id,choice_id,question_weight,feedback_weight,weight,created_at,updated_at) VALUES (gen_random_uuid(),?,?,0,?,LEAST(?,GREATEST(?,?)),NOW(),NOW()) ON CONFLICT (profile_id,choice_id) DO UPDATE SET feedback_weight=EXCLUDED.feedback_weight,weight=LEAST(?,GREATEST(?," + table + ".question_weight+EXCLUDED.feedback_weight)),updated_at=NOW()"
		if err := db.Exec(q, profileID, row.ChoiceID, row.Weight, maximum, minimum, row.Weight, maximum, minimum).Error; err != nil {
			return entity.ErrInternal
		}
	}
	return nil
}

func recalculateDiscovery(db *gorm.DB, userID uuid.UUID, like, maybe, skip, minimum, maximum float64) error {
	returnError := db.Exec(`UPDATE user_profiles SET discovery_adjustment=LEAST(?,GREATEST(?,COALESCE((SELECT SUM(CASE f.feedback_type WHEN 'LIKE' THEN ? WHEN 'MAYBE' THEN ? WHEN 'SKIP' THEN ? ELSE 0 END) FROM feedbacks f JOIN recommendation_details rd ON rd.id=f.recommendation_detail_id JOIN recommendations r ON r.id=rd.recommendation_id WHERE f.user_id=? AND (rd.recommendation_type='TRY_SOMETHING_NEW' OR r.source='DAILY_DISCOVERY')),0))),updated_at=NOW() WHERE user_id=?`, maximum, minimum, like, maybe, skip, userID, userID)
	if returnError.Error != nil {
		return entity.ErrInternal
	}
	if returnError.RowsAffected == 0 {
		return entity.ErrUserProfileRequired
	}
	return nil
}
