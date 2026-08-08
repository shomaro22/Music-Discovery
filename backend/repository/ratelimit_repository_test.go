package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRateLimitRepositoryUsesFullRefillTTL(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	repo := NewRateLimitRepository(client)
	result, err := repo.Allow(context.Background(), "rate_limit:refresh:ip:test", 30.0/300.0, 30, 1, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Allowed || result.Remaining != 29 {
		t.Fatalf("result=%+v", result)
	}
	ttl := mini.TTL("rate_limit:refresh:ip:test")
	if ttl != 300*time.Second {
		t.Fatalf("ttl=%s", ttl)
	}
}

func TestRateLimitRepositoryReturnsRedisError(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: time.Millisecond, ReadTimeout: time.Millisecond, WriteTimeout: time.Millisecond, MaxRetries: 0})
	t.Cleanup(func() { _ = client.Close() })
	_, err := NewRateLimitRepository(client).Allow(context.Background(), "key", 1, 1, 1, time.Now())
	if err == nil {
		t.Fatal("expected Redis error")
	}
}
