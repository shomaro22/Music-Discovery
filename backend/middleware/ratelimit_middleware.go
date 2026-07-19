package middleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type RateLimitRule struct {
	Capacity        int
	RefillPerSecond float64
	TTL             time.Duration
}

type RateLimitConfig struct {
	Login   RateLimitRule
	SignUp  RateLimitRule
	Refresh RateLimitRule
}

func LoadRateLimitConfigFromEnv() (RateLimitConfig, error) {
	login, err := loadRule("LOGIN", 20, 20.0/60.0, 120*time.Second)
	if err != nil {
		return RateLimitConfig{}, err
	}
	signup, err := loadRule("SIGNUP", 10, 10.0/60.0, 120*time.Second)
	if err != nil {
		return RateLimitConfig{}, err
	}
	refresh, err := loadRule("REFRESH", 30, 6.0/60.0, 300*time.Second)
	if err != nil {
		return RateLimitConfig{}, err
	}
	return RateLimitConfig{Login: login, SignUp: signup, Refresh: refresh}, nil
}

func loadRule(scope string, capacityDefault int, rateDefault float64, ttlDefault time.Duration) (RateLimitRule, error) {
	capacity, err := envInt([]string{"RATE_LIMIT_" + scope + "_CAPACITY", "RATE_LIMIT_CAPACITY"}, capacityDefault)
	if err != nil || capacity <= 0 {
		return RateLimitRule{}, fmt.Errorf("invalid %s rate limit capacity", scope)
	}
	rate, err := envFloat([]string{"RATE_LIMIT_" + scope + "_RATE", "RATE_LIMIT_RATE"}, rateDefault)
	if err != nil || rate <= 0 {
		return RateLimitRule{}, fmt.Errorf("invalid %s rate limit rate", scope)
	}
	ttlSeconds, err := envInt([]string{"RATE_LIMIT_" + scope + "_TTL_SECONDS", "RATE_LIMIT_TTL_SECONDS"}, int(ttlDefault.Seconds()))
	if err != nil || ttlSeconds <= 0 {
		return RateLimitRule{}, fmt.Errorf("invalid %s rate limit ttl", scope)
	}
	return RateLimitRule{Capacity: capacity, RefillPerSecond: rate, TTL: time.Duration(ttlSeconds) * time.Second}, nil
}

func envInt(keys []string, fallback int) (int, error) {
	for _, key := range keys {
		if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
			return strconv.Atoi(raw)
		}
	}
	return fallback, nil
}
func envFloat(keys []string, fallback float64) (float64, error) {
	for _, key := range keys {
		if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
			return strconv.ParseFloat(raw, 64)
		}
	}
	return fallback, nil
}

type IRateLimitMiddleware interface {
	SignUp(next echo.HandlerFunc) echo.HandlerFunc
	Login(next echo.HandlerFunc) echo.HandlerFunc
	Refresh(next echo.HandlerFunc) echo.HandlerFunc
}

type rateLimitMiddleware struct {
	redisClient *redis.Client
	logger      *slog.Logger
	config      RateLimitConfig
}

func NewRateLimitMiddleware(redisClient *redis.Client, optional ...any) IRateLimitMiddleware {
	logger := slog.Default()
	cfg, _ := LoadRateLimitConfigFromEnv()
	for _, value := range optional {
		switch typed := value.(type) {
		case *slog.Logger:
			logger = typed
		case RateLimitConfig:
			cfg = typed
		}
	}
	return &rateLimitMiddleware{redisClient: redisClient, logger: logger, config: cfg}
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
if tokens >= cost then tokens = tokens - cost allowed = 1 else retry_after = math.ceil((cost - tokens) / refill_rate) end
redis.call('HSET', key, 'tokens', tokens, 'updated_at', now)
redis.call('EXPIRE', key, ttl)
return {allowed, retry_after}
`)

func (m *rateLimitMiddleware) SignUp(next echo.HandlerFunc) echo.HandlerFunc {
	return m.withTokenBucket("signup", m.config.SignUp, next)
}
func (m *rateLimitMiddleware) Login(next echo.HandlerFunc) echo.HandlerFunc {
	return m.withTokenBucket("login", m.config.Login, next)
}
func (m *rateLimitMiddleware) Refresh(next echo.HandlerFunc) echo.HandlerFunc {
	return m.withTokenBucket("refresh", m.config.Refresh, next)
}

func (m *rateLimitMiddleware) withTokenBucket(scope string, rule RateLimitRule, next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		key := fmt.Sprintf("rate_limit:%s:%s", scope, c.RealIP())
		result, err := tokenBucketScript.Run(c.Request().Context(), m.redisClient, []string{key}, rule.Capacity, rule.RefillPerSecond, float64(time.Now().UnixMilli())/1000, 1, int(rule.TTL.Seconds())).Slice()
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			m.logger.Error("rate_limit_redis_error", "scope", scope, "request_id", c.Response().Header().Get(echo.HeaderXRequestID), "error", err.Error())
			return next(c)
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
