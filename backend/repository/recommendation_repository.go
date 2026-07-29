package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"music-discovery/backend/entity"
)

type RecommendationFeedbackSignals struct {
	ArtistWeights   map[uuid.UUID]float64
	SongWeights     map[uuid.UUID]float64
	ArtistPenalties map[uuid.UUID]float64
	SongPenalties   map[uuid.UUID]float64
}

type RecommendationCandidateCondition struct {
	ExcludedSongIDs []uuid.UUID
	Limit           int
}

type IRecommendationRepository interface {
	FindCandidates(ctx context.Context, condition RecommendationCandidateCondition) ([]entity.RecommendationSongCandidate, error)
	FindRecommendedSongIDs(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]uuid.UUID, error)
	Create(ctx context.Context, tx ITransaction, recommendation *entity.Recommendation) error
	FindLatest(ctx context.Context, userID uuid.UUID, source entity.RecommendationSource) (*entity.Recommendation, error)
	FindByID(ctx context.Context, recommendationID uuid.UUID) (*entity.Recommendation, error)
	HasRecentNormal(ctx context.Context, userID uuid.UUID, since time.Time) (bool, error)
	CountByArtist(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]int64, error)
	CountBySong(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]int64, error)
	FindFeedbackSignals(ctx context.Context, userID uuid.UUID) (RecommendationFeedbackSignals, error)
}

type recommendationRepository struct{ db *gorm.DB }

func NewRecommendationRepository(db *gorm.DB) IRecommendationRepository {
	return &recommendationRepository{db: db}
}

func (r *recommendationRepository) FindCandidates(ctx context.Context, condition RecommendationCandidateCondition) ([]entity.RecommendationSongCandidate, error) {
	limit := condition.Limit
	if limit <= 0 {
		limit = 100
	}
	query := r.db.WithContext(ctx).Model(&entity.Song{}).
		Joins("JOIN artists ON artists.id = songs.artist_id AND artists.deleted_at IS NULL").
		Where("songs.deleted_at IS NULL AND songs.spotify_url <> ''").
		Where("EXISTS (SELECT 1 FROM song_tags st JOIN tags t ON t.id = st.tag_id AND t.deleted_at IS NULL WHERE st.song_id = songs.id AND LOWER(t.category) = 'genre')")
	if len(condition.ExcludedSongIDs) > 0 {
		query = query.Where("songs.id NOT IN ?", condition.ExcludedSongIDs)
	}
	var songs []entity.Song
	if err := query.Order("songs.popularity DESC, songs.id ASC").Limit(limit).Find(&songs).Error; err != nil {
		return nil, entity.ErrInternal
	}
	if len(songs) == 0 {
		return []entity.RecommendationSongCandidate{}, nil
	}
	songIDs := make([]uuid.UUID, 0, len(songs))
	artistIDs := make([]uuid.UUID, 0, len(songs))
	for _, song := range songs {
		songIDs = append(songIDs, song.ID)
		artistIDs = append(artistIDs, song.ArtistID)
	}
	var artists []entity.Artist
	if err := r.db.WithContext(ctx).Where("id IN ?", artistIDs).Find(&artists).Error; err != nil {
		return nil, entity.ErrInternal
	}
	artistMap := map[uuid.UUID]entity.Artist{}
	for _, artist := range artists {
		artistMap[artist.ID] = artist
	}
	type tagRow struct {
		OwnerID              uuid.UUID
		ID                   uuid.UUID
		Name, Slug, Category string
	}
	var songTagRows, artistTagRows []tagRow
	if err := r.db.WithContext(ctx).Table("song_tags st").Select("st.song_id owner_id, t.id, t.name, t.slug, t.category").Joins("JOIN tags t ON t.id = st.tag_id AND t.deleted_at IS NULL").Where("st.song_id IN ?", songIDs).Scan(&songTagRows).Error; err != nil {
		return nil, entity.ErrInternal
	}
	if err := r.db.WithContext(ctx).Table("artist_tags at").Select("at.artist_id owner_id, t.id, t.name, t.slug, t.category").Joins("JOIN tags t ON t.id = at.tag_id AND t.deleted_at IS NULL").Where("at.artist_id IN ?", artistIDs).Scan(&artistTagRows).Error; err != nil {
		return nil, entity.ErrInternal
	}
	songTags := map[uuid.UUID][]entity.Tag{}
	artistTags := map[uuid.UUID][]entity.Tag{}
	for _, row := range songTagRows {
		songTags[row.OwnerID] = append(songTags[row.OwnerID], entity.Tag{ID: row.ID, Name: row.Name, Slug: row.Slug, Category: row.Category})
	}
	for _, row := range artistTagRows {
		artistTags[row.OwnerID] = append(artistTags[row.OwnerID], entity.Tag{ID: row.ID, Name: row.Name, Slug: row.Slug, Category: row.Category})
	}
	result := make([]entity.RecommendationSongCandidate, 0, len(songs))
	for _, song := range songs {
		result = append(result, entity.RecommendationSongCandidate{Song: song, Artist: artistMap[song.ArtistID], SongTags: songTags[song.ID], ArtistTags: artistTags[song.ArtistID]})
	}
	return result, nil
}

func (r *recommendationRepository) FindRecommendedSongIDs(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.WithContext(ctx).Table("recommendation_details rd").Distinct("rd.song_id").Joins("JOIN recommendations r ON r.id = rd.recommendation_id").Where("r.user_id = ? AND r.recommended_at >= ? AND r.recommended_at <= ?", userID, from, to).Pluck("rd.song_id", &ids).Error
	if err != nil {
		return nil, entity.ErrInternal
	}
	return ids, nil
}
func (r *recommendationRepository) Create(ctx context.Context, tx ITransaction, recommendation *entity.Recommendation) error {
	db, ok := transactionDB(tx)
	if !ok || recommendation == nil {
		return entity.ErrInvalidInput
	}
	if err := db.WithContext(ctx).Create(recommendation).Error; err != nil {
		return entity.ErrRecommendationSaveFailed
	}
	return nil
}
func (r *recommendationRepository) preload(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Preload("Details", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).Preload("Details.Song.Artist")
}
func (r *recommendationRepository) FindLatest(ctx context.Context, userID uuid.UUID, source entity.RecommendationSource) (*entity.Recommendation, error) {
	var value entity.Recommendation
	err := r.preload(ctx).Where("user_id = ? AND source = ?", userID, source).Order("recommended_at DESC").First(&value).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, entity.ErrRecommendationNotFound
	}
	if err != nil {
		return nil, entity.ErrInternal
	}
	return &value, nil
}
func (r *recommendationRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Recommendation, error) {
	var value entity.Recommendation
	err := r.preload(ctx).First(&value, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, entity.ErrRecommendationNotFound
	}
	if err != nil {
		return nil, entity.ErrInternal
	}
	return &value, nil
}
func (r *recommendationRepository) HasRecentNormal(ctx context.Context, userID uuid.UUID, since time.Time) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Recommendation{}).Where("user_id = ? AND source = ? AND recommended_at >= ?", userID, entity.RecommendationSourceNormal, since).Count(&count).Error
	return count > 0, err
}
func (r *recommendationRepository) CountByArtist(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]int64, error) {
	type row struct {
		ArtistID uuid.UUID
		Count    int64
	}
	var rows []row
	err := r.db.WithContext(ctx).Table("recommendation_details rd").Select("s.artist_id, COUNT(*) count").Joins("JOIN recommendations r ON r.id = rd.recommendation_id").Joins("JOIN songs s ON s.id = rd.song_id").Where("r.user_id = ?", userID).Group("s.artist_id").Scan(&rows).Error
	result := map[uuid.UUID]int64{}
	for _, v := range rows {
		result[v.ArtistID] = v.Count
	}
	return result, err
}
func (r *recommendationRepository) CountBySong(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]int64, error) {
	type row struct {
		SongID uuid.UUID
		Count  int64
	}
	var rows []row
	err := r.db.WithContext(ctx).Table("recommendation_details rd").Select("rd.song_id, COUNT(*) count").Joins("JOIN recommendations r ON r.id = rd.recommendation_id").Where("r.user_id = ?", userID).Group("rd.song_id").Scan(&rows).Error
	result := map[uuid.UUID]int64{}
	for _, v := range rows {
		result[v.SongID] = v.Count
	}
	return result, err
}

func (r *recommendationRepository) FindFeedbackSignals(ctx context.Context, userID uuid.UUID) (RecommendationFeedbackSignals, error) {
	type row struct {
		ArtistID, SongID uuid.UUID
		Status           entity.FeedbackStatus
	}
	var rows []row
	err := r.db.WithContext(ctx).Table("recommendation_details rd").
		Select("s.artist_id, rd.song_id, rd.feedback_status status").
		Joins("JOIN recommendations r ON r.id = rd.recommendation_id").
		Joins("JOIN songs s ON s.id = rd.song_id").
		Where("r.user_id = ? AND rd.feedback_status IS NOT NULL", userID).Scan(&rows).Error
	if err != nil {
		return RecommendationFeedbackSignals{}, entity.ErrInternal
	}
	result := RecommendationFeedbackSignals{ArtistWeights: map[uuid.UUID]float64{}, SongWeights: map[uuid.UUID]float64{}, ArtistPenalties: map[uuid.UUID]float64{}, SongPenalties: map[uuid.UUID]float64{}}
	for _, value := range rows {
		switch value.Status {
		case entity.FeedbackStatusLike:
			result.ArtistWeights[value.ArtistID] += 1
			result.SongWeights[value.SongID] += 1
		case entity.FeedbackStatusMaybe:
			result.ArtistWeights[value.ArtistID] += .25
			result.SongWeights[value.SongID] += .25
		case entity.FeedbackStatusSkip:
			result.ArtistWeights[value.ArtistID] -= .5
			result.SongWeights[value.SongID] -= .5
			result.ArtistPenalties[value.ArtistID] += 1
			result.SongPenalties[value.SongID] += 1.5
		}
	}
	return result, nil
}
