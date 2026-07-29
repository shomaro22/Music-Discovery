package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"music-discovery/backend/repository"
)

type RateLimitInput struct{ Scope, IP, Email, UserID string }

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
	repository repository.IRateLimitRepository
	clock      IClock
}

func NewRateLimitUsecase(repo repository.IRateLimitRepository, clock IClock) IRateLimitUsecase {
	return &rateLimitUsecase{repository: repo, clock: clock}
}

func (u *rateLimitUsecase) Allow(ctx context.Context, input RateLimitInput) (RateLimitDecision, error) {
	policies, err := policiesFor(input)
	if err != nil {
		return RateLimitDecision{}, err
	}
	var last RateLimitDecision
	for _, policy := range policies {
		result, allowErr := u.repository.Allow(ctx, policy.key, policy.rate, policy.capacity, 1, u.clock.Now())
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
	email := strings.ToLower(strings.TrimSpace(input.Email))
	userID := strings.TrimSpace(input.UserID)
	switch input.Scope {
	case "signup":
		policies := []rateLimitPolicy{{name: "signup_ip", key: fmt.Sprintf("rate_limit:signup:ip:%s", ip), capacity: 10, rate: 10.0 / 60.0}}
		if email != "" {
			policies = append(policies, rateLimitPolicy{name: "signup_email", key: fmt.Sprintf("rate_limit:signup:email:%s", email), capacity: 10, rate: 10.0 / 60.0})
		}
		return policies, nil
	case "login":
		policies := []rateLimitPolicy{{name: "login_ip", key: fmt.Sprintf("rate_limit:login:ip:%s", ip), capacity: 20, rate: 20.0 / 60.0}}
		if email != "" {
			policies = append(policies, rateLimitPolicy{name: "login_email", key: fmt.Sprintf("rate_limit:login:email:%s", email), capacity: 20, rate: 20.0 / 60.0})
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
	default:
		return nil, fmt.Errorf("unknown rate limit scope: %s", input.Scope)
	}
}
