package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type IRecommendationLockRepository interface {
	Acquire(ctx context.Context, userID uuid.UUID, ttl time.Duration) (string, bool, error)
	Release(ctx context.Context, userID uuid.UUID, token string) error
}

type recommendationLockRepository struct{ client *redis.Client }

func NewRecommendationLockRepository(client *redis.Client) IRecommendationLockRepository {
	return &recommendationLockRepository{client: client}
}
func recommendationLockKey(userID uuid.UUID) string {
	return fmt.Sprintf("recommendation:generation:%s", userID.String())
}
func (r *recommendationLockRepository) Acquire(ctx context.Context, userID uuid.UUID, ttl time.Duration) (string, bool, error) {
	token := uuid.NewString()
	ok, err := r.client.SetNX(ctx, recommendationLockKey(userID), token, ttl).Result()
	return token, ok, err
}

var releaseRecommendationLockScript = redis.NewScript(`if redis.call("GET", KEYS[1]) == ARGV[1] then return redis.call("DEL", KEYS[1]) else return 0 end`)

func (r *recommendationLockRepository) Release(ctx context.Context, userID uuid.UUID, token string) error {
	return releaseRecommendationLockScript.Run(ctx, r.client, []string{recommendationLockKey(userID)}, token).Err()
}
