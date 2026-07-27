package migrate

import (
	"os"
	"testing"
	"time"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func openQuestionMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; PostgreSQL integration test skipped")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	if err := CreateUsers(db); err != nil {
		t.Fatalf("create users: %v", err)
	}
	if err := CreateQuestionTables(db); err != nil {
		t.Fatalf("create question tables: %v", err)
	}
	if err := db.Exec("TRUNCATE TABLE user_answer_choices, user_answers, question_answer_sessions, user_genre_weights, user_element_weights, user_atmosphere_weights, user_profiles, question_choices, questions, users CASCADE").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if err := CreateQuestionTables(db); err != nil {
		t.Fatalf("reseed: %v", err)
	}
	return db
}

func TestQuestionSeedRerunPreservesManualInactiveState(t *testing.T) {
	db := openQuestionMigrationTestDB(t)
	var question entity.Question
	if err := db.Where("code = ?", entity.QuestionCodeMood).First(&question).Error; err != nil {
		t.Fatal(err)
	}
	var choice entity.QuestionChoice
	if err := db.Where("question_id = ?", question.ID).First(&choice).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&question).Update("is_active", false).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&choice).Update("is_active", false).Error; err != nil {
		t.Fatal(err)
	}
	if err := CreateQuestionTables(db); err != nil {
		t.Fatalf("rerun migration: %v", err)
	}
	if err := db.First(&question, "id = ?", question.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&choice, "id = ?", choice.ID).Error; err != nil {
		t.Fatal(err)
	}
	if question.IsActive || choice.IsActive {
		t.Fatalf("seed rerun reactivated records: question=%v choice=%v", question.IsActive, choice.IsActive)
	}
}

func TestQuestionWeightMigrationBackfillsExistingWeight(t *testing.T) {
	db := openQuestionMigrationTestDB(t)
	now := time.Now().UTC()
	user := entity.User{ID: uuid.New(), Name: "Migration", Email: "migration-" + uuid.NewString() + "@example.com", PasswordHash: "hash", Role: entity.RoleUser, Status: entity.UserStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	var choice entity.QuestionChoice
	if err := db.Joins("JOIN questions ON questions.id = question_choices.question_id").Where("questions.code = ?", entity.QuestionCodeGenre).First(&choice).Error; err != nil {
		t.Fatal(err)
	}
	profile := entity.UserProfile{ID: uuid.New(), UserID: user.ID, DiscoveryLevel: "DISCOVERY_MEDIUM", CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&profile).Error; err != nil {
		t.Fatal(err)
	}
	weight := entity.UserGenreWeight{ID: uuid.New(), ProfileID: profile.ID, ChoiceID: choice.ID, Weight: 1.4, QuestionWeight: 0, FeedbackWeight: 0, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&weight).Error; err != nil {
		t.Fatal(err)
	}
	if err := CreateQuestionTables(db); err != nil {
		t.Fatalf("migrate existing weights: %v", err)
	}
	if err := db.First(&weight, "id = ?", weight.ID).Error; err != nil {
		t.Fatal(err)
	}
	if weight.QuestionWeight != 1.4 || weight.FeedbackWeight != 0 || weight.Weight != 1.4 {
		t.Fatalf("question=%v feedback=%v weight=%v", weight.QuestionWeight, weight.FeedbackWeight, weight.Weight)
	}
}
