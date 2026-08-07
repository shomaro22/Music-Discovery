package migrate

import (
	testutil "music-discovery/backend/test"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func openRecommendationMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testutil.OpenTestDB(t)
	if err := CreateUsers(db); err != nil {
		t.Fatalf("create users: %v", err)
	}
	if err := CreateAdminTables(db); err != nil {
		t.Fatalf("create admin tables: %v", err)
	}
	if err := CreateRecommendationTables(db); err != nil {
		t.Fatalf("create recommendation tables: %v", err)
	}
	return db
}

func TestRecommendationMigrationReplacesLegacyPositionIndex(t *testing.T) {
	db := openRecommendationMigrationTestDB(t)
	if err := db.Exec(`DROP INDEX IF EXISTS uq_recommendation_position`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE UNIQUE INDEX uq_recommendation_position ON recommendation_details (recommendation_id, recommendation_type, position)`).Error; err != nil {
		t.Fatal(err)
	}

	if err := CreateRecommendationTables(db); err != nil {
		t.Fatalf("rerun recommendation migration: %v", err)
	}

	var definition string
	if err := db.Raw(`SELECT indexdef FROM pg_indexes WHERE schemaname = current_schema() AND indexname = 'uq_recommendation_position'`).Scan(&definition).Error; err != nil {
		t.Fatal(err)
	}
	normalized := strings.ToLower(strings.Join(strings.Fields(definition), " "))
	if !strings.Contains(normalized, "(recommendation_id, \"position\")") && !strings.Contains(normalized, "(recommendation_id, position)") {
		t.Fatalf("unexpected position index definition: %s", definition)
	}
	if strings.Contains(normalized, "recommendation_type") {
		t.Fatalf("legacy recommendation_type column remains in index: %s", definition)
	}
}

func TestRecommendationMigrationCreatesUserForeignKey(t *testing.T) {
	db := openRecommendationMigrationTestDB(t)
	var count int64
	err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.table_constraints tc
		JOIN information_schema.constraint_column_usage ccu
		  ON tc.constraint_name = ccu.constraint_name
		 AND tc.constraint_schema = ccu.constraint_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
		  AND tc.table_schema = current_schema()
		  AND tc.table_name = 'recommendations'
		  AND ccu.table_name = 'users'
		  AND ccu.column_name = 'id'
	`).Scan(&count).Error
	if err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("recommendations.user_id foreign key to users.id was not created")
	}
}
