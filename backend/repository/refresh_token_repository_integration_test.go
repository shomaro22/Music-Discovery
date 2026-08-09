package repository

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
)

func setupRefreshTokenRepositoryTest(t *testing.T) (*refreshTokenRepository, *entity.User) {
	t.Helper()
	db := openTestPostgres(t)
	if err := db.AutoMigrate(&entity.RefreshToken{}); err != nil {
		t.Fatalf("migrate refresh_tokens: %v", err)
	}
	if err := db.Exec("TRUNCATE TABLE refresh_tokens, users RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	user := validUser(uuid.NewString() + "@example.com")
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return &refreshTokenRepository{db: db}, user
}

func newStoredRefreshToken(userID, familyID uuid.UUID, hash string, now time.Time) *entity.RefreshToken {
	return &entity.RefreshToken{ID: uuid.New(), UserID: userID, TokenHash: hash, FamilyID: familyID, ExpiresAt: now.Add(time.Hour), CreatedAt: now.Add(-time.Minute), UpdatedAt: now.Add(-time.Minute)}
}

func TestRefreshTokenRepositoryFindByTokenHash(t *testing.T) {
	repo, user := setupRefreshTokenRepositoryTest(t)
	now := time.Now().UTC()
	stored := newStoredRefreshToken(user.ID, uuid.New(), "find-hash", now)
	if err := repo.db.Create(stored).Error; err != nil {
		t.Fatal(err)
	}
	found, err := repo.FindByTokenHash(context.Background(), "find-hash")
	if err != nil || found.ID != stored.ID {
		t.Fatalf("found=%+v err=%v", found, err)
	}
	if _, err := repo.FindByTokenHash(context.Background(), "missing-hash"); !errors.Is(err, entity.ErrInvalidRefreshToken) {
		t.Fatalf("err=%v", err)
	}
}

func TestRefreshTokenRepositoryRotate(t *testing.T) {
	repo, user := setupRefreshTokenRepositoryTest(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	familyID := uuid.New()
	current := newStoredRefreshToken(user.ID, familyID, "rotate-old", now)
	if err := repo.db.Create(current).Error; err != nil {
		t.Fatal(err)
	}
	next := newStoredRefreshToken(user.ID, familyID, "rotate-new", now)
	if err := repo.Rotate(context.Background(), current.ID, next, now); err != nil {
		t.Fatal(err)
	}
	var updated entity.RefreshToken
	if err := repo.db.First(&updated, "id = ?", current.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.UsedAt == nil || updated.RevokedAt == nil || updated.ReplacedByID == nil || *updated.ReplacedByID != next.ID {
		t.Fatalf("updated=%+v", updated)
	}
	var created entity.RefreshToken
	if err := repo.db.First(&created, "id = ?", next.ID).Error; err != nil {
		t.Fatal(err)
	}
	if created.FamilyID != familyID {
		t.Fatalf("family=%s", created.FamilyID)
	}
}

func TestRefreshTokenRepositoryRotateRollback(t *testing.T) {
	repo, user := setupRefreshTokenRepositoryTest(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	familyID := uuid.New()
	current := newStoredRefreshToken(user.ID, familyID, "rollback-old", now)
	conflict := newStoredRefreshToken(user.ID, familyID, "duplicate-hash", now)
	if err := repo.db.Create(current).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.db.Create(conflict).Error; err != nil {
		t.Fatal(err)
	}
	next := newStoredRefreshToken(user.ID, familyID, "duplicate-hash", now)
	if err := repo.Rotate(context.Background(), current.ID, next, now); !errors.Is(err, entity.ErrInternal) {
		t.Fatalf("err=%v", err)
	}
	var stored entity.RefreshToken
	if err := repo.db.First(&stored, "id = ?", current.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.RevokedAt != nil || stored.ReplacedByID != nil {
		t.Fatalf("rotation was not rolled back: %+v", stored)
	}
}

func TestRefreshTokenRepositoryConcurrentRotateOnlyOneSucceeds(t *testing.T) {
	repo, user := setupRefreshTokenRepositoryTest(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	familyID := uuid.New()
	current := newStoredRefreshToken(user.ID, familyID, "concurrent-old", now)
	if err := repo.db.Create(current).Error; err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			next := newStoredRefreshToken(user.ID, familyID, "concurrent-new-"+uuid.NewString(), now)
			errs <- repo.Rotate(context.Background(), current.ID, next, now.Add(time.Duration(i)*time.Microsecond))
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)
	success, reused := 0, 0
	for err := range errs {
		switch {
		case err == nil:
			success++
		case errors.Is(err, entity.ErrRefreshTokenReused):
			reused++
		default:
			t.Fatalf("unexpected err=%v", err)
		}
	}
	if success != 1 || reused != 1 {
		t.Fatalf("success=%d reused=%d", success, reused)
	}
	var count int64
	if err := repo.db.Model(&entity.RefreshToken{}).Where("family_id = ?", familyID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("family token count=%d", count)
	}
}

func TestRefreshTokenRepositoryRevokeFamily(t *testing.T) {
	repo, user := setupRefreshTokenRepositoryTest(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	targetFamily := uuid.New()
	otherFamily := uuid.New()

	targetOne := newStoredRefreshToken(user.ID, targetFamily, "logout-target-1", now)
	targetTwo := newStoredRefreshToken(user.ID, targetFamily, "logout-target-2", now)
	alreadyRevoked := newStoredRefreshToken(user.ID, targetFamily, "logout-target-revoked", now)
	previousRevokedAt := now.Add(-time.Hour)
	alreadyRevoked.RevokedAt = &previousRevokedAt
	other := newStoredRefreshToken(user.ID, otherFamily, "logout-other", now)

	for _, token := range []*entity.RefreshToken{targetOne, targetTwo, alreadyRevoked, other} {
		if err := repo.db.Create(token).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := repo.RevokeFamily(context.Background(), targetFamily, now); err != nil {
		t.Fatal(err)
	}

	var tokens []entity.RefreshToken
	if err := repo.db.Where("id IN ?", []uuid.UUID{targetOne.ID, targetTwo.ID, alreadyRevoked.ID, other.ID}).Find(&tokens).Error; err != nil {
		t.Fatal(err)
	}
	byID := make(map[uuid.UUID]entity.RefreshToken, len(tokens))
	for _, token := range tokens {
		byID[token.ID] = token
	}
	if byID[targetOne.ID].RevokedAt == nil || byID[targetTwo.ID].RevokedAt == nil {
		t.Fatalf("target family was not fully revoked: %+v", byID)
	}
	if byID[alreadyRevoked.ID].RevokedAt == nil || !byID[alreadyRevoked.ID].RevokedAt.Equal(previousRevokedAt) {
		t.Fatalf("already revoked token was changed: %+v", byID[alreadyRevoked.ID])
	}
	if byID[other.ID].RevokedAt != nil {
		t.Fatalf("other family was revoked: %+v", byID[other.ID])
	}

	var userAfter entity.User
	if err := repo.db.First(&userAfter, "id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if userAfter.TokenVersion != user.TokenVersion {
		t.Fatalf("token version changed: before=%d after=%d", user.TokenVersion, userAfter.TokenVersion)
	}

	if err := repo.RevokeFamily(context.Background(), uuid.New(), now); err != nil {
		t.Fatalf("missing family must be idempotent: %v", err)
	}
}
