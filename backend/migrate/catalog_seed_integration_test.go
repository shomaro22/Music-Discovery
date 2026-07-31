package migrate

import (
	"os"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"music-discovery/backend/entity"
)

func openCatalogSeedTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; PostgreSQL integration test skipped")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	return db
}

func TestCatalogSeedsAreIdempotent(t *testing.T) {
	db := openCatalogSeedTestDB(t)
	if err := CreateAdminTables(db); err != nil {
		t.Fatalf("create admin tables: %v", err)
	}
	if err := db.Exec("TRUNCATE TABLE song_tags, artist_tags, songs, artists, tags CASCADE").Error; err != nil {
		t.Fatalf("truncate catalog: %v", err)
	}
	legacyTag := entity.Tag{ID: uuid.New(), Name: "ポップ", Slug: "pop", Category: "genre"}
	if err := db.Create(&legacyTag).Error; err != nil {
		t.Fatalf("create legacy tag: %v", err)
	}
	if err := CreateAdminTables(db); err != nil {
		t.Fatalf("reseed tags: %v", err)
	}
	var migratedTag entity.Tag
	if err := db.First(&migratedTag, "id = ?", legacyTag.ID).Error; err != nil {
		t.Fatalf("reload migrated legacy tag: %v", err)
	}
	if migratedTag.Slug != "genre-pop" {
		t.Fatalf("legacy tag slug = %s, want genre-pop", migratedTag.Slug)
	}
	if err := SeedDevelopmentCatalogForEnvironment(db, "development"); err != nil {
		t.Fatalf("seed development catalog: %v", err)
	}
	if err := SeedDevelopmentCatalogForEnvironment(db, "development"); err != nil {
		t.Fatalf("rerun development catalog seed: %v", err)
	}

	assertCatalogCount(t, db, &entity.Tag{}, 26)
	assertCatalogCount(t, db, &entity.Artist{}, int64(len(developmentCatalogSeeds)))
	assertCatalogCount(t, db, &entity.Song{}, int64(len(developmentCatalogSeeds)))
	assertCatalogCount(t, db, &entity.SongTag{}, int64(len(developmentCatalogSeeds)*10))
}

func assertCatalogCount(t *testing.T, db *gorm.DB, model interface{}, expected int64) {
	t.Helper()
	var count int64
	if err := db.Model(model).Count(&count).Error; err != nil {
		t.Fatalf("count %T: %v", model, err)
	}
	if count != expected {
		t.Fatalf("%T count = %d, want %d", model, count, expected)
	}
}
