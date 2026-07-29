package repository

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestRecommendationLockAllowsOnlyOneConcurrentAcquirePerUser(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	repo := NewRecommendationLockRepository(client)
	userID := uuid.New()

	start := make(chan struct{})
	results := make(chan bool, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, acquired, err := repo.Acquire(context.Background(), userID, time.Minute)
			if err != nil {
				t.Errorf("acquire: %v", err)
				return
			}
			results <- acquired
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	acquired := 0
	for result := range results {
		if result {
			acquired++
		}
	}
	if acquired != 1 {
		t.Fatalf("acquired=%d want=1", acquired)
	}
}

func TestRecommendationLockDoesNotReleaseAnotherToken(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	repo := NewRecommendationLockRepository(client)
	userID := uuid.New()

	token, acquired, err := repo.Acquire(context.Background(), userID, time.Minute)
	if err != nil || !acquired {
		t.Fatalf("acquire=%v err=%v", acquired, err)
	}
	if err := repo.Release(context.Background(), userID, "different-token"); err != nil {
		t.Fatal(err)
	}
	if got, err := client.Get(context.Background(), recommendationLockKey(userID)).Result(); err != nil || got != token {
		t.Fatalf("lock token=%q err=%v want=%q", got, err, token)
	}
	if err := repo.Release(context.Background(), userID, token); err != nil {
		t.Fatal(err)
	}
	if client.Exists(context.Background(), recommendationLockKey(userID)).Val() != 0 {
		t.Fatal("matching token must release the lock")
	}
}

func TestRecommendationLockUsesProvidedTTL(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	repo := NewRecommendationLockRepository(client)
	userID := uuid.New()

	_, acquired, err := repo.Acquire(context.Background(), userID, 2*time.Minute)
	if err != nil || !acquired {
		t.Fatalf("acquire=%v err=%v", acquired, err)
	}
	ttl := mini.TTL(recommendationLockKey(userID))
	if ttl != 2*time.Minute {
		t.Fatalf("ttl=%v want=%v", ttl, 2*time.Minute)
	}
}
