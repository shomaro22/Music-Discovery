package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/repository"

	"github.com/google/uuid"
)

type FeedbackDelta struct{ Genre, Artist, Song, Profile, Discovery float64 }
type FeedbackConfig struct {
	Like, Maybe, Skip                                      FeedbackDelta
	MinimumWeight, MaximumWeight                           float64
	MinimumDiscoveryAdjustment, MaximumDiscoveryAdjustment float64
}
type SaveFeedbackInput struct {
	RecommendationDetailID uuid.UUID
	FeedbackType           entity.FeedbackType
}
type FeedbackResult struct {
	Feedback *entity.Feedback
	Created  bool
}
type FeedbackDetail struct {
	RecommendationDetailID uuid.UUID
	Status                 string
	UpdatedAt              *time.Time
}

type IFeedbackUsecase interface {
	Save(context.Context, uuid.UUID, SaveFeedbackInput) (*FeedbackResult, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (*FeedbackDetail, error)
}
type feedbackUsecase struct {
	users    repository.IUserRepository
	feedback repository.IFeedbackRepository
	tx       repository.ITransactionManager
	clock    IClock
	config   FeedbackConfig
}

func NewFeedbackUsecase(users repository.IUserRepository, feedback repository.IFeedbackRepository, tx repository.ITransactionManager, clock IClock, config FeedbackConfig) IFeedbackUsecase {
	return &feedbackUsecase{users: users, feedback: feedback, tx: tx, clock: clock, config: config}
}

func (u *feedbackUsecase) validateUser(ctx context.Context, userID uuid.UUID) error {
	user, err := u.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.Status == entity.UserStatusSuspended {
		return entity.ErrUserSuspended
	}
	return nil
}

func (u *feedbackUsecase) Save(ctx context.Context, userID uuid.UUID, input SaveFeedbackInput) (*FeedbackResult, error) {
	if userID == uuid.Nil || input.RecommendationDetailID == uuid.Nil {
		return nil, entity.ErrInvalidRecommendationDetailID
	}
	if !input.FeedbackType.IsValid() {
		return nil, entity.ErrInvalidFeedbackType
	}
	if err := u.validateUser(ctx, userID); err != nil {
		return nil, err
	}
	var result FeedbackResult
	err := u.tx.WithinTransaction(ctx, func(tx repository.ITransaction) error {
		profile, err := u.feedback.FindLearningProfileForUpdate(ctx, tx, userID)
		if err != nil {
			return err
		}
		target, err := u.feedback.FindTargetForUpdate(ctx, tx, input.RecommendationDetailID)
		if err != nil {
			return err
		}
		if target.Recommendation.UserID != userID {
			return entity.ErrFeedbackAccessDenied
		}
		existing, findErr := u.feedback.FindByDetailID(ctx, tx, input.RecommendationDetailID)
		if findErr != nil && !errors.Is(findErr, entity.ErrNotFound) {
			return findErr
		}
		created := errors.Is(findErr, entity.ErrNotFound)
		if !created && existing.Type == input.FeedbackType {
			result = FeedbackResult{Feedback: existing, Created: false}
			return nil
		}
		now := u.clock.Now().UTC()
		if created {
			existing = &entity.Feedback{UserID: userID, RecommendationDetailID: input.RecommendationDetailID, SongID: target.Song.ID, ArtistID: target.Artist.ID, Type: input.FeedbackType, CreatedAt: now, UpdatedAt: now}
			if err := u.feedback.Create(ctx, tx, existing); err != nil {
				return err
			}
		} else {
			existing.Type = input.FeedbackType
			existing.UpdatedAt = now
			if err := u.feedback.Update(ctx, tx, existing); err != nil {
				return err
			}
		}
		sources, err := u.feedback.ListLearningSources(ctx, tx, userID)
		if err != nil {
			return err
		}
		snapshot := u.calculateLearningSnapshot(userID, profile, sources, now)
		if err := u.feedback.ReplaceLearningSnapshot(ctx, tx, userID, profile.ID, snapshot, now); err != nil {
			return err
		}
		result = FeedbackResult{Feedback: existing, Created: created}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (u *feedbackUsecase) Get(ctx context.Context, userID, detailID uuid.UUID) (*FeedbackDetail, error) {
	if detailID == uuid.Nil {
		return nil, entity.ErrInvalidRecommendationDetailID
	}
	if err := u.validateUser(ctx, userID); err != nil {
		return nil, err
	}
	target, err := u.feedback.FindTarget(ctx, detailID)
	if err != nil {
		return nil, err
	}
	if target.Recommendation.UserID != userID {
		return nil, entity.ErrFeedbackAccessDenied
	}
	result := &FeedbackDetail{RecommendationDetailID: detailID, Status: "NONE"}
	err = u.tx.WithinTransaction(ctx, func(tx repository.ITransaction) error {
		value, findErr := u.feedback.FindByDetailID(ctx, tx, detailID)
		if errors.Is(findErr, entity.ErrNotFound) {
			return nil
		}
		if findErr != nil {
			return findErr
		}
		result.Status = string(value.Type)
		updated := value.UpdatedAt
		result.UpdatedAt = &updated
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (u *feedbackUsecase) calculateLearningSnapshot(userID uuid.UUID, profile *entity.UserProfile, sources []entity.FeedbackLearningSource, now time.Time) entity.FeedbackLearningSnapshot {
	artistScores := make(map[uuid.UUID]float64)
	songScores := make(map[uuid.UUID]float64)
	genreScores := make(map[uuid.UUID]float64)
	elementScores := make(map[uuid.UUID]float64)
	atmosphereScores := make(map[uuid.UUID]float64)
	discovery := 0.0

	for _, source := range sources {
		delta := u.feedbackDelta(source.FeedbackType)
		artistScores[source.ArtistID] += delta.Artist
		songScores[source.SongID] += delta.Song
		if source.Exploration {
			discovery += delta.Discovery
		}
		seen := make(map[uuid.UUID]struct{}, len(source.Tags))
		for _, tag := range source.Tags {
			if _, exists := seen[tag.ChoiceID]; exists {
				continue
			}
			seen[tag.ChoiceID] = struct{}{}
			switch strings.ToLower(strings.TrimSpace(tag.Category)) {
			case "genre":
				genreScores[tag.ChoiceID] += delta.Genre
			case "element", "feature":
				elementScores[tag.ChoiceID] += delta.Profile
			case "atmosphere", "scene":
				atmosphereScores[tag.ChoiceID] += delta.Profile
			}
		}
	}

	snapshot := entity.FeedbackLearningSnapshot{
		DiscoveryAdjustment: clamp(discovery, u.config.MinimumDiscoveryAdjustment, u.config.MaximumDiscoveryAdjustment),
	}
	for id, score := range artistScores {
		snapshot.ArtistWeights = append(snapshot.ArtistWeights, entity.UserArtistWeight{ID: uuid.New(), UserID: userID, ArtistID: id, Weight: clamp(score, u.config.MinimumWeight, u.config.MaximumWeight), CreatedAt: now, UpdatedAt: now})
	}
	for id, score := range songScores {
		snapshot.SongWeights = append(snapshot.SongWeights, entity.UserSongWeight{ID: uuid.New(), UserID: userID, SongID: id, Weight: clamp(score, u.config.MinimumWeight, u.config.MaximumWeight), CreatedAt: now, UpdatedAt: now})
	}
	snapshot.GenreWeights = buildGenreWeights(profile.ID, profile.GenreWeights, genreScores, now, u.config.MinimumWeight, u.config.MaximumWeight)
	snapshot.ElementWeights = buildElementWeights(profile.ID, profile.ElementWeights, elementScores, now, u.config.MinimumWeight, u.config.MaximumWeight)
	snapshot.AtmosphereWeights = buildAtmosphereWeights(profile.ID, profile.AtmosphereWeights, atmosphereScores, now, u.config.MinimumWeight, u.config.MaximumWeight)
	return snapshot
}

func (u *feedbackUsecase) feedbackDelta(value entity.FeedbackType) FeedbackDelta {
	switch value {
	case entity.FeedbackTypeLike:
		return u.config.Like
	case entity.FeedbackTypeMaybe:
		return u.config.Maybe
	default:
		return u.config.Skip
	}
}

func clamp(value, minimum, maximum float64) float64 {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func buildGenreWeights(profileID uuid.UUID, current []entity.UserGenreWeight, feedback map[uuid.UUID]float64, now time.Time, minimum, maximum float64) []entity.UserGenreWeight {
	values := make(map[uuid.UUID]entity.UserGenreWeight, len(current)+len(feedback))
	for _, value := range current {
		values[value.ChoiceID] = value
	}
	for choiceID := range feedback {
		if _, exists := values[choiceID]; !exists {
			values[choiceID] = entity.UserGenreWeight{ID: uuid.New(), ProfileID: profileID, ChoiceID: choiceID, CreatedAt: now}
		}
	}
	result := make([]entity.UserGenreWeight, 0, len(values))
	for choiceID, value := range values {
		value.ProfileID = profileID
		value.FeedbackWeight = clamp(feedback[choiceID], minimum, maximum)
		value.Weight = clamp(value.QuestionWeight+value.FeedbackWeight, minimum, maximum)
		value.UpdatedAt = now
		result = append(result, value)
	}
	return result
}

func buildElementWeights(profileID uuid.UUID, current []entity.UserElementWeight, feedback map[uuid.UUID]float64, now time.Time, minimum, maximum float64) []entity.UserElementWeight {
	values := make(map[uuid.UUID]entity.UserElementWeight, len(current)+len(feedback))
	for _, value := range current {
		values[value.ChoiceID] = value
	}
	for choiceID := range feedback {
		if _, exists := values[choiceID]; !exists {
			values[choiceID] = entity.UserElementWeight{ID: uuid.New(), ProfileID: profileID, ChoiceID: choiceID, CreatedAt: now}
		}
	}
	result := make([]entity.UserElementWeight, 0, len(values))
	for choiceID, value := range values {
		value.ProfileID = profileID
		value.FeedbackWeight = clamp(feedback[choiceID], minimum, maximum)
		value.Weight = clamp(value.QuestionWeight+value.FeedbackWeight, minimum, maximum)
		value.UpdatedAt = now
		result = append(result, value)
	}
	return result
}

func buildAtmosphereWeights(profileID uuid.UUID, current []entity.UserAtmosphereWeight, feedback map[uuid.UUID]float64, now time.Time, minimum, maximum float64) []entity.UserAtmosphereWeight {
	values := make(map[uuid.UUID]entity.UserAtmosphereWeight, len(current)+len(feedback))
	for _, value := range current {
		values[value.ChoiceID] = value
	}
	for choiceID := range feedback {
		if _, exists := values[choiceID]; !exists {
			values[choiceID] = entity.UserAtmosphereWeight{ID: uuid.New(), ProfileID: profileID, ChoiceID: choiceID, CreatedAt: now}
		}
	}
	result := make([]entity.UserAtmosphereWeight, 0, len(values))
	for choiceID, value := range values {
		value.ProfileID = profileID
		value.FeedbackWeight = clamp(feedback[choiceID], minimum, maximum)
		value.Weight = clamp(value.QuestionWeight+value.FeedbackWeight, minimum, maximum)
		value.UpdatedAt = now
		result = append(result, value)
	}
	return result
}
