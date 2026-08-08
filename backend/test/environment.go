package testutil

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const TestDatabaseName = "music_discovery_test"

// OpenTestDB opens only the dedicated PostgreSQL test database.
// In CI, a missing TEST_DATABASE_URL is a configuration error and fails the test.
func OpenTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		if isCI() {
			t.Fatal("TEST_DATABASE_URL is required in CI")
		}
		t.Skip("TEST_DATABASE_URL is not set; PostgreSQL integration test skipped")
	}

	if err := validateTestDSN(dsn); err != nil {
		t.Fatal(err)
	}

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	assertTestDatabase(t, database)
	return database
}

// ResetDatabase truncates application tables while preserving the migrated schema.
// It re-checks the actual connected database before any destructive statement.
func ResetDatabase(t *testing.T, database *gorm.DB) {
	t.Helper()
	if database == nil {
		t.Fatal("database is nil")
	}
	assertTestDatabase(t, database)

	var tables []string
	if err := database.Raw(`SELECT tablename FROM pg_tables WHERE schemaname = 'public'`).Scan(&tables).Error; err != nil {
		t.Fatalf("list test tables: %v", err)
	}
	if len(tables) == 0 {
		return
	}

	quoted := make([]string, 0, len(tables))
	for _, table := range tables {
		quoted = append(quoted, `"`+strings.ReplaceAll(table, `"`, `""`)+`"`)
	}
	if err := database.Exec("TRUNCATE TABLE " + strings.Join(quoted, ",") + " RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatalf("truncate test database: %v", err)
	}
}

func validateTestDSN(dsn string) error {
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("parse TEST_DATABASE_URL: %w", err)
	}
	if config.Database != TestDatabaseName {
		return fmt.Errorf("refusing to use database %q; expected exactly %q", config.Database, TestDatabaseName)
	}
	return nil
}

func assertTestDatabase(t *testing.T, database *gorm.DB) {
	t.Helper()
	var name string
	if err := database.Raw("SELECT current_database()").Scan(&name).Error; err != nil {
		t.Fatalf("read current database: %v", err)
	}
	if name != TestDatabaseName {
		t.Fatalf("refusing destructive test operation on database %q; expected exactly %q", name, TestDatabaseName)
	}
}

func isCI() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("CI")))
	return value == "1" || value == "true" || value == "yes"
}
