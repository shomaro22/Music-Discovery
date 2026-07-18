package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type IRateLimitMiddleware interface {
	SignUp(next echo.HandlerFunc) echo.HandlerFunc
}

type rateLimitMiddleware struct {
	redisClient *redis.Client
}

func NewRateLimitMiddleware(redisClient *redis.Client) IRateLimitMiddleware {
	return &rateLimitMiddleware{redisClient: redisClient}
}

var tokenBucketScript = redis.NewScript(`
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local cost = tonumber(ARGV[4])
local ttl = tonumber(ARGV[5])

local values = redis.call('HMGET', key, 'tokens', 'updated_at')
local tokens = tonumber(values[1])
local updated_at = tonumber(values[2])
if tokens == nil then tokens = capacity end
if updated_at == nil then updated_at = now end

local elapsed = math.max(0, now - updated_at)
tokens = math.min(capacity, tokens + elapsed * refill_rate)

local allowed = 0
local retry_after = 0
if tokens >= cost then
  tokens = tokens - cost
  allowed = 1
else
  retry_after = math.ceil((cost - tokens) / refill_rate)
end

redis.call('HSET', key, 'tokens', tokens, 'updated_at', now)
redis.call('EXPIRE', key, ttl)
return {allowed, retry_after}
`)

func (m *rateLimitMiddleware) SignUp(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		const capacity = 10
		const refillPerSecond = 10.0 / 60.0
		key := fmt.Sprintf("rate_limit:signup:%s", c.RealIP())

		result, err := tokenBucketScript.Run(
			c.Request().Context(), m.redisClient, []string{key},
			capacity, refillPerSecond, float64(time.Now().UnixMilli())/1000, 1, 120,
		).Slice()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}

		allowed, _ := result[0].(int64)
		if allowed == 1 {
			return next(c)
		}

		retryAfter, _ := result[1].(int64)
		if retryAfter < 1 {
			retryAfter = 1
		}
		c.Response().Header().Set("Retry-After", strconv.FormatInt(retryAfter, 10))
		return echo.NewHTTPError(http.StatusTooManyRequests)
	}
}
