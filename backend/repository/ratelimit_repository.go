package repository

import (
	"context"
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimitResult struct {
	Allowed    bool
	Remaining  float64
	RetryAfter time.Duration
}

type IRateLimitRepository interface {
	Allow(ctx context.Context, key string, rate float64, capacity, cost int, now time.Time) (RateLimitResult, error)
}

type rateLimitRepository struct{ client *redis.Client }

func NewRateLimitRepository(client *redis.Client) IRateLimitRepository {
	return &rateLimitRepository{client: client}
}

var tokenBucketScript = redis.NewScript(`
local values = redis.call('HMGET', KEYS[1], 'tokens', 'updated_at')
local capacity = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local cost = tonumber(ARGV[4])
local ttl = tonumber(ARGV[5])
local tokens = tonumber(values[1]) or capacity
local updated = tonumber(values[2]) or now
tokens = math.min(capacity, tokens + math.max(0, now-updated)*rate)
local allowed = 0
local retry = 0
if tokens >= cost then
  tokens = tokens-cost
  allowed = 1
else
  retry = math.ceil((cost-tokens)/rate)
end
redis.call('HSET', KEYS[1], 'tokens', tokens, 'updated_at', now)
redis.call('EXPIRE', KEYS[1], ttl)
return {tostring(allowed), tostring(tokens), tostring(retry)}`)

func (r *rateLimitRepository) Allow(ctx context.Context, key string, rate float64, capacity, cost int, now time.Time) (RateLimitResult, error) {
	if r.client == nil || key == "" || rate <= 0 || capacity <= 0 || cost <= 0 {
		return RateLimitResult{}, errors.New("invalid rate limit repository input")
	}
	ttlSeconds := int(math.Ceil(float64(capacity) / rate))
	if ttlSeconds < 1 {
		ttlSeconds = 1
	}
	result, err := tokenBucketScript.Run(
		ctx,
		r.client,
		[]string{key},
		capacity,
		rate,
		float64(now.UnixMilli())/1000,
		cost,
		ttlSeconds,
	).StringSlice()
	if err != nil {
		return RateLimitResult{}, err
	}
	if len(result) != 3 {
		return RateLimitResult{}, errors.New("invalid rate limit script result")
	}
	allowedValue, err := strconv.ParseInt(result[0], 10, 64)
	if err != nil {
		return RateLimitResult{}, errors.New("invalid rate limit allowed result")
	}
	remaining, err := strconv.ParseFloat(result[1], 64)
	if err != nil {
		return RateLimitResult{}, errors.New("invalid rate limit remaining result")
	}
	retry, err := strconv.ParseInt(result[2], 10, 64)
	if err != nil {
		return RateLimitResult{}, errors.New("invalid rate limit retry result")
	}
	allowed := allowedValue == 1
	if !allowed && retry < 1 {
		retry = 1
	}
	return RateLimitResult{Allowed: allowed, Remaining: remaining, RetryAfter: time.Duration(retry) * time.Second}, nil
}
