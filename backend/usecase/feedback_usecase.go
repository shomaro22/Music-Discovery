package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"music-discovery/backend/entity"
	"music-discovery/backend/repository"
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
	target, err := u.feedback.FindTarget(ctx, input.RecommendationDetailID)
	if err != nil {
		return nil, err
	}
	if target.Recommendation.UserID != userID {
		return nil, entity.ErrFeedbackAccessDenied
	}
	if target.Song.ID == uuid.Nil {
		return nil, entity.ErrNotFound
	}
	if target.Artist.ID == uuid.Nil {
		return nil, entity.ErrNotFound
	}

	var result FeedbackResult
	err = u.tx.WithinTransaction(ctx, func(tx repository.ITransaction) error {
		existing, findErr := u.feedback.FindByDetailID(ctx, tx, input.RecommendationDetailID)
		if findErr != nil && !errors.Is(findErr, entity.ErrNotFound) {
			return findErr
		}
		created := errors.Is(findErr, entity.ErrNotFound)
		if !created && existing.Type == input.FeedbackType {
			result = FeedbackResult{Feedback: existing, Created: false}
			return nil
		}
		if !created {
			if applyErr := u.apply(ctx, tx, userID, target, existing.Type, -1); applyErr != nil {
				return applyErr
			}
		}
		if applyErr := u.apply(ctx, tx, userID, target, input.FeedbackType, 1); applyErr != nil {
			return applyErr
		}
		now := u.clock.Now().UTC()
		if created {
			existing = &entity.Feedback{UserID: userID, RecommendationDetailID: input.RecommendationDetailID, SongID: target.Song.ID, ArtistID: target.Artist.ID, Type: input.FeedbackType, CreatedAt: now, UpdatedAt: now}
			if createErr := u.feedback.Create(ctx, tx, existing); createErr != nil {
				return createErr
			}
		} else {
			existing.Type = input.FeedbackType
			existing.UpdatedAt = now
			if updateErr := u.feedback.Update(ctx, tx, existing); updateErr != nil {
				return updateErr
			}
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
func (u *feedbackUsecase) delta(value entity.FeedbackType) FeedbackDelta {
	switch value {
	case entity.FeedbackTypeLike:
		return u.config.Like
	case entity.FeedbackTypeMaybe:
		return u.config.Maybe
	default:
		return u.config.Skip
	}
}
func (u *feedbackUsecase) apply(ctx context.Context, tx repository.ITransaction, userID uuid.UUID, target *repository.FeedbackTarget, feedbackType entity.FeedbackType, direction float64) error {
	delta := u.delta(feedbackType)
	if err := u.feedback.ApplyArtistWeight(ctx, tx, userID, target.Artist.ID, delta.Artist*direction, u.config.MinimumWeight, u.config.MaximumWeight); err != nil {
		return err
	}
	if err := u.feedback.ApplySongWeight(ctx, tx, userID, target.Song.ID, delta.Song*direction, u.config.MinimumWeight, u.config.MaximumWeight); err != nil {
		return err
	}
	seen := make(map[uuid.UUID]struct{}, len(target.Tags))
	for _, tag := range target.Tags {
		if _, ok := seen[tag.ID]; ok {
			continue
		}
		seen[tag.ID] = struct{}{}
		category := strings.ToLower(strings.TrimSpace(tag.Category))
		tagDelta := delta.Profile
		if category == "genre" {
			tagDelta = delta.Genre
		}
		if category != "genre" && category != "element" && category != "atmosphere" {
			continue
		}
		if err := u.feedback.ApplyProfileTagWeight(ctx, tx, userID, tag, tagDelta*direction, u.config.MinimumWeight, u.config.MaximumWeight); err != nil {
			return err
		}
	}
	if target.Detail.Type == entity.RecommendationTypeTrySomethingNew || target.Recommendation.Source == entity.RecommendationSourceDailyDiscovery {
		if err := u.feedback.ApplyDiscoveryAdjustment(ctx, tx, userID, delta.Discovery*direction, u.config.MinimumDiscoveryAdjustment, u.config.MaximumDiscoveryAdjustment); err != nil {
			return err
		}
	}
	return nil
}
