package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"music-discovery/backend/repository"
)

type RateLimitInput struct{ Scope, IP, UserID, Email string }

type RateLimitDecision struct {
	Allowed    bool
	Remaining  float64
	RetryAfter time.Duration
	LimitName  string
}

type IRateLimitUsecase interface {
	Allow(ctx context.Context, input RateLimitInput) (RateLimitDecision, error)
}

type rateLimitUsecase struct {
	repository         repository.IRateLimitRepository
	clock              IClock
	capacityMultiplier int
}

func NewRateLimitUsecase(repo repository.IRateLimitRepository, clock IClock) IRateLimitUsecase {
	return NewRateLimitUsecaseWithCapacityMultiplier(repo, clock, 1)
}

func NewRateLimitUsecaseWithCapacityMultiplier(repo repository.IRateLimitRepository, clock IClock, capacityMultiplier int) IRateLimitUsecase {
	if capacityMultiplier < 1 {
		capacityMultiplier = 1
	}
	return &rateLimitUsecase{repository: repo, clock: clock, capacityMultiplier: capacityMultiplier}
}

func (u *rateLimitUsecase) Allow(ctx context.Context, input RateLimitInput) (RateLimitDecision, error) {
	policies, err := policiesFor(input)
	if err != nil {
		return RateLimitDecision{}, err
	}
	var last RateLimitDecision
	for _, policy := range policies {
		result, allowErr := u.repository.Allow(ctx, policy.key, policy.rate*float64(u.capacityMultiplier), policy.capacity*u.capacityMultiplier, 1, u.clock.Now())
		if allowErr != nil {
			return RateLimitDecision{}, allowErr
		}
		last = RateLimitDecision{Allowed: result.Allowed, Remaining: result.Remaining, RetryAfter: result.RetryAfter, LimitName: policy.name}
		if !result.Allowed {
			return last, nil
		}
	}
	return last, nil
}

type rateLimitPolicy struct {
	name     string
	key      string
	rate     float64
	capacity int
}

func policiesFor(input RateLimitInput) ([]rateLimitPolicy, error) {
	ip := strings.TrimSpace(input.IP)
	userID := strings.TrimSpace(input.UserID)
	emailDigest := rateLimitEmailDigest(input.Email)
	switch input.Scope {
	case "signup":
		policies := []rateLimitPolicy{{name: "signup_ip", key: fmt.Sprintf("rate_limit:signup:ip:%s", ip), capacity: 10, rate: 10.0 / 60.0}}
		if emailDigest != "" {
			policies = append(policies, rateLimitPolicy{name: "signup_email", key: "rate_limit:signup:email:" + emailDigest, capacity: 5, rate: 5.0 / 60.0})
		}
		return policies, nil
	case "login":
		policies := []rateLimitPolicy{{name: "login_ip", key: fmt.Sprintf("rate_limit:login:ip:%s", ip), capacity: 20, rate: 20.0 / 60.0}}
		if emailDigest != "" {
			policies = append(policies, rateLimitPolicy{name: "login_email", key: "rate_limit:login:email:" + emailDigest, capacity: 10, rate: 10.0 / 300.0})
		}
		return policies, nil
	case "refresh":
		return []rateLimitPolicy{{name: "refresh_ip", key: fmt.Sprintf("rate_limit:refresh:ip:%s", ip), capacity: 30, rate: 30.0 / 300.0}}, nil
	case "withdraw":
		return []rateLimitPolicy{{name: "withdraw_ip", key: fmt.Sprintf("rate_limit:withdraw:ip:%s", ip), capacity: 5, rate: 5.0 / 600.0}}, nil
	case "question_answers":
		return []rateLimitPolicy{{name: "question_answers_ip", key: fmt.Sprintf("rate_limit:question_answers:ip:%s", ip), capacity: 10, rate: 10.0 / 60.0}}, nil
	case "recommendations":
		policies := []rateLimitPolicy{{name: "recommendations_ip", key: fmt.Sprintf("rate_limit:recommendations:ip:%s", ip), capacity: 5, rate: 5.0 / 60.0}}
		if userID != "" {
			policies = append(policies, rateLimitPolicy{name: "recommendations_user", key: fmt.Sprintf("rate_limit:recommendations:user:%s", userID), capacity: 5, rate: 5.0 / 60.0})
		}
		return policies, nil
	case "daily_discovery":
		policies := []rateLimitPolicy{{name: "daily_discovery_ip", key: fmt.Sprintf("rate_limit:daily_discovery:ip:%s", ip), capacity: 5, rate: 5.0 / 60.0}}
		if userID != "" {
			policies = append(policies, rateLimitPolicy{name: "daily_discovery_user", key: fmt.Sprintf("rate_limit:daily_discovery:user:%s", userID), capacity: 5, rate: 5.0 / 60.0})
		}
		return policies, nil
	case "feedback":
		policies := []rateLimitPolicy{{name: "feedback_ip", key: fmt.Sprintf("rate_limit:feedback:ip:%s", ip), capacity: 30, rate: 30.0 / 60.0}}
		if userID != "" {
			policies = append(policies, rateLimitPolicy{name: "feedback_user", key: fmt.Sprintf("rate_limit:feedback:user:%s", userID), capacity: 30, rate: 30.0 / 60.0})
		}
		return policies, nil
	default:
		return nil, fmt.Errorf("unknown rate limit scope: %s", input.Scope)
	}
}

func rateLimitEmailDigest(email string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}
