package usecase

import (
	"context"
	"testing"
	"time"

	"music-discovery/backend/repository"
)

type rateLimitRepositoryCall struct {
	key      string
	rate     float64
	capacity int
	cost     int
}

type fakeRateLimitRepository struct {
	calls   []rateLimitRepositoryCall
	results []repository.RateLimitResult
	err     error
}

func (f *fakeRateLimitRepository) Allow(_ context.Context, key string, rate float64, capacity, cost int, _ time.Time) (repository.RateLimitResult, error) {
	f.calls = append(f.calls, rateLimitRepositoryCall{key: key, rate: rate, capacity: capacity, cost: cost})
	if f.err != nil {
		return repository.RateLimitResult{}, f.err
	}
	result := f.results[len(f.calls)-1]
	return result, nil
}

type fixedRateLimitClock struct{ now time.Time }

func (c fixedRateLimitClock) Now() time.Time { return c.now }

func TestRateLimitUsecaseLoginChecksIPThenEmail(t *testing.T) {
	repo := &fakeRateLimitRepository{results: []repository.RateLimitResult{
		{Allowed: true, Remaining: 19},
		{Allowed: true, Remaining: 18},
	}}
	u := NewRateLimitUsecase(repo, fixedRateLimitClock{now: time.Unix(100, 0)})
	decision, err := u.Allow(context.Background(), RateLimitInput{Scope: "login", IP: "192.0.2.10", Email: " User@Example.COM "})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.calls) != 2 {
		t.Fatalf("calls=%d", len(repo.calls))
	}
	if repo.calls[0].key != "rate_limit:login:ip:192.0.2.10" {
		t.Fatalf("ip key=%s", repo.calls[0].key)
	}
	if repo.calls[1].key != "rate_limit:login:email:user@example.com" {
		t.Fatalf("email key=%s", repo.calls[1].key)
	}
	if !decision.Allowed || decision.LimitName != "login_email" || decision.Remaining != 18 {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestRateLimitUsecaseStopsWhenIPLimitIsDenied(t *testing.T) {
	repo := &fakeRateLimitRepository{results: []repository.RateLimitResult{{Allowed: false, Remaining: 0, RetryAfter: 3 * time.Second}}}
	u := NewRateLimitUsecase(repo, fixedRateLimitClock{})
	decision, err := u.Allow(context.Background(), RateLimitInput{Scope: "signup", IP: "192.0.2.20", Email: "user@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.calls) != 1 {
		t.Fatalf("calls=%d", len(repo.calls))
	}
	if decision.Allowed || decision.LimitName != "signup_ip" || decision.RetryAfter != 3*time.Second {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestRateLimitUsecaseRefreshPolicy(t *testing.T) {
	repo := &fakeRateLimitRepository{results: []repository.RateLimitResult{{Allowed: false, Remaining: 0, RetryAfter: 12 * time.Second}}}
	u := NewRateLimitUsecase(repo, fixedRateLimitClock{})
	decision, err := u.Allow(context.Background(), RateLimitInput{Scope: "refresh", IP: "203.0.113.10"})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.calls) != 1 {
		t.Fatalf("calls=%d", len(repo.calls))
	}
	call := repo.calls[0]
	if call.key != "rate_limit:refresh:ip:203.0.113.10" || call.capacity != 30 || call.rate != 30.0/300.0 || call.cost != 1 {
		t.Fatalf("call=%+v", call)
	}
	if decision.Allowed || decision.LimitName != "refresh_ip" || decision.RetryAfter != 12*time.Second {
		t.Fatalf("decision=%+v", decision)
	}
}
