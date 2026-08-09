package usecase

import (
	"context"
	"testing"
	"time"

	"music-discovery/backend/entity"
)

type rateLimitRepositoryCall struct {
	key      string
	rate     float64
	capacity int
	cost     int
}

type fakeRateLimitRepository struct {
	calls   []rateLimitRepositoryCall
	results []entity.RateLimitResult
	err     error
}

func (f *fakeRateLimitRepository) Allow(_ context.Context, key string, rate float64, capacity, cost int, _ time.Time) (entity.RateLimitResult, error) {
	f.calls = append(f.calls, rateLimitRepositoryCall{key: key, rate: rate, capacity: capacity, cost: cost})
	if f.err != nil {
		return entity.RateLimitResult{}, f.err
	}
	result := f.results[len(f.calls)-1]
	return result, nil
}

type fixedRateLimitClock struct{ now time.Time }

func (c fixedRateLimitClock) Now() time.Time { return c.now }

func TestRateLimitUsecaseLoginUsesIPOnly(t *testing.T) {
	repo := &fakeRateLimitRepository{results: []entity.RateLimitResult{{Allowed: true, Remaining: 19}}}
	u := NewRateLimitUsecase(repo, fixedRateLimitClock{now: time.Unix(100, 0)})
	decision, err := u.Allow(context.Background(), RateLimitInput{Scope: "login", IP: "192.0.2.10"})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.calls) != 1 {
		t.Fatalf("calls=%d", len(repo.calls))
	}
	if repo.calls[0].key != "rate_limit:login:ip:192.0.2.10" {
		t.Fatalf("ip key=%s", repo.calls[0].key)
	}
	if !decision.Allowed || decision.LimitName != "login_ip" || decision.Remaining != 19 {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestRateLimitUsecaseStopsWhenIPLimitIsDenied(t *testing.T) {
	repo := &fakeRateLimitRepository{results: []entity.RateLimitResult{{Allowed: false, Remaining: 0, RetryAfter: 3 * time.Second}}}
	u := NewRateLimitUsecase(repo, fixedRateLimitClock{})
	decision, err := u.Allow(context.Background(), RateLimitInput{Scope: "signup", IP: "192.0.2.20"})
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
	repo := &fakeRateLimitRepository{results: []entity.RateLimitResult{{Allowed: false, Remaining: 0, RetryAfter: 12 * time.Second}}}
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

func TestRateLimitUsecaseCapacityMultiplierScalesPerformancePolicy(t *testing.T) {
	repo := &fakeRateLimitRepository{results: []entity.RateLimitResult{{Allowed: true, Remaining: 4999}}}
	u := NewRateLimitUsecaseWithCapacityMultiplier(repo, fixedRateLimitClock{}, 1000)
	_, err := u.Allow(context.Background(), RateLimitInput{Scope: "recommendations", IP: "192.0.2.30"})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.calls) != 1 {
		t.Fatalf("calls=%d", len(repo.calls))
	}
	call := repo.calls[0]
	if call.capacity != 5000 {
		t.Fatalf("capacity=%d want=5000", call.capacity)
	}
	if call.rate != (5.0/60.0)*1000 {
		t.Fatalf("rate=%f want=%f", call.rate, (5.0/60.0)*1000)
	}
}

func TestRateLimitUsecaseLoginUsesIPThenEmailDigest(t *testing.T) {
	repo := &fakeRateLimitRepository{results: []entity.RateLimitResult{
		{Allowed: true, Remaining: 19},
		{Allowed: true, Remaining: 9},
	}}
	u := NewRateLimitUsecase(repo, fixedRateLimitClock{now: time.Unix(100, 0)})
	decision, err := u.Allow(context.Background(), RateLimitInput{Scope: "login", IP: "192.0.2.10", Email: "  User@Example.COM "})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.calls) != 2 {
		t.Fatalf("calls=%d", len(repo.calls))
	}
	if repo.calls[0].key != "rate_limit:login:ip:192.0.2.10" {
		t.Fatalf("ip key=%s", repo.calls[0].key)
	}
	wantDigest := rateLimitEmailDigest("user@example.com")
	if repo.calls[1].key != "rate_limit:login:email:"+wantDigest {
		t.Fatalf("email key=%s", repo.calls[1].key)
	}
	if decision.LimitName != "login_email" || !decision.Allowed {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestRateLimitUsecaseSignupStopsBeforeEmailWhenIPDenied(t *testing.T) {
	repo := &fakeRateLimitRepository{results: []entity.RateLimitResult{{Allowed: false, Remaining: 0, RetryAfter: time.Second}}}
	u := NewRateLimitUsecase(repo, fixedRateLimitClock{})
	decision, err := u.Allow(context.Background(), RateLimitInput{Scope: "signup", IP: "192.0.2.20", Email: "user@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.calls) != 1 || decision.LimitName != "signup_ip" {
		t.Fatalf("calls=%d decision=%+v", len(repo.calls), decision)
	}
}
