package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/migrate"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func openQuestionTestPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	db := openTestPostgres(t)
	if err := migrate.CreateQuestionTables(db); err != nil {
		t.Fatalf("migrate question tables: %v", err)
	}
	if err := db.Exec("TRUNCATE TABLE user_answer_choices, user_answers, question_answer_sessions, user_genre_weights, user_element_weights, user_atmosphere_weights, user_profiles, question_choices, questions, users CASCADE").Error; err != nil {
		t.Fatalf("truncate question tables: %v", err)
	}
	if err := migrate.CreateQuestionTables(db); err != nil {
		t.Fatalf("reseed question tables: %v", err)
	}
	return db
}

func createQuestionTestUser(t *testing.T, db *gorm.DB) *entity.User {
	t.Helper()
	user := validUser("question-" + uuid.NewString() + "@example.com")
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func TestQuestionRepositoryExcludesInactiveQuestionAndChoice(t *testing.T) {
	db := openQuestionTestPostgres(t)
	var question entity.Question
	if err := db.Where("code = ?", entity.QuestionCodeGenre).First(&question).Error; err != nil {
		t.Fatal(err)
	}
	var inactiveChoice entity.QuestionChoice
	if err := db.Where("question_id = ?", question.ID).Order("sort_order").First(&inactiveChoice).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&inactiveChoice).Update("is_active", false).Error; err != nil {
		t.Fatal(err)
	}
	questions, err := NewQuestionRepository(db).ListActive(context.Background())
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	for _, item := range questions {
		for _, choice := range item.Choices {
			if choice.ID == inactiveChoice.ID {
				t.Fatal("inactive choice was returned")
			}
		}
	}
	if err := db.Model(&question).Update("is_active", false).Error; err != nil {
		t.Fatal(err)
	}
	questions, err = NewQuestionRepository(db).ListActive(context.Background())
	if err != nil {
		t.Fatalf("ListActive after question disable: %v", err)
	}
	for _, item := range questions {
		if item.ID == question.ID {
			t.Fatal("inactive question was returned")
		}
	}
}

func TestQuestionAnswerRepositoryLatestAndTransactionRollback(t *testing.T) {
	db := openQuestionTestPostgres(t)
	user := createQuestionTestUser(t, db)
	var question entity.Question
	if err := db.Preload("Choices").Where("code = ?", entity.QuestionCodeMood).First(&question).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewQuestionAnswerRepository(db)
	txManager := NewAuthTransactionManager(db)
	oldTime := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	newTime := oldTime.Add(time.Hour)
	for _, answeredAt := range []time.Time{oldTime, newTime} {
		session := &entity.QuestionAnswerSession{ID: uuid.New(), UserID: user.ID, AnsweredAt: answeredAt, CreatedAt: answeredAt}
		session.Answers = []entity.UserAnswer{{ID: uuid.New(), AnswerSessionID: session.ID, QuestionID: question.ID, CreatedAt: answeredAt, Choices: []entity.UserAnswerChoice{{ID: uuid.New(), QuestionChoiceID: question.Choices[0].ID, CreatedAt: answeredAt}}}}
		if err := txManager.WithinTransaction(context.Background(), func(tx ITransaction) error { return repo.CreateSession(context.Background(), tx, session) }); err != nil {
			t.Fatalf("create session: %v", err)
		}
	}
	latest, err := repo.GetLatest(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if !latest.AnsweredAt.Equal(newTime) {
		t.Fatalf("answered_at=%v want=%v", latest.AnsweredAt, newTime)
	}

	rollbackID := uuid.New()
	err = txManager.WithinTransaction(context.Background(), func(tx ITransaction) error {
		session := &entity.QuestionAnswerSession{ID: rollbackID, UserID: user.ID, AnsweredAt: newTime.Add(time.Hour), CreatedAt: newTime.Add(time.Hour)}
		if err := repo.CreateSession(context.Background(), tx, session); err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	if err == nil {
		t.Fatal("expected rollback error")
	}
	var count int64
	if err := db.Model(&entity.QuestionAnswerSession{}).Where("id = ?", rollbackID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("rolled back session count=%d", count)
	}
}

func TestUserProfileResetPreservesFeedbackAndClearsInactiveQuestionWeight(t *testing.T) {
	db := openQuestionTestPostgres(t)
	user := createQuestionTestUser(t, db)
	var choice entity.QuestionChoice
	if err := db.Joins("JOIN questions ON questions.id = question_choices.question_id").Where("questions.code = ?", entity.QuestionCodeGenre).First(&choice).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	profile := &entity.UserProfile{ID: uuid.New(), UserID: user.ID, DiscoveryLevel: "DISCOVERY_MEDIUM", CreatedAt: now, UpdatedAt: now}
	if err := db.Create(profile).Error; err != nil {
		t.Fatal(err)
	}
	weight := entity.UserGenreWeight{ID: uuid.New(), ProfileID: profile.ID, ChoiceID: choice.ID, QuestionWeight: 1, FeedbackWeight: 0.7, Weight: 1.7, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&weight).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&choice).Update("is_active", false).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewUserProfileRepository(db)
	txManager := NewAuthTransactionManager(db)
	if err := txManager.WithinTransaction(context.Background(), func(tx ITransaction) error {
		return repo.ResetQuestionWeights(context.Background(), tx, profile.ID, now.Add(time.Minute))
	}); err != nil {
		t.Fatalf("reset: %v", err)
	}
	var got entity.UserGenreWeight
	if err := db.First(&got, "id = ?", weight.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.QuestionWeight != 0 || got.FeedbackWeight != 0.7 || got.Weight != 0.7 {
		t.Fatalf("weights question=%v feedback=%v total=%v", got.QuestionWeight, got.FeedbackWeight, got.Weight)
	}
}

func TestQuestionForeignKeysRejectUnknownReferences(t *testing.T) {
	db := openQuestionTestPostgres(t)
	now := time.Now().UTC()
	profile := entity.UserProfile{ID: uuid.New(), UserID: uuid.New(), DiscoveryLevel: "DISCOVERY_MEDIUM", CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&profile).Error; err == nil {
		t.Fatal("expected user_profiles.user_id foreign-key violation")
	}
	answer := entity.UserAnswer{ID: uuid.New(), AnswerSessionID: uuid.New(), QuestionID: uuid.New(), CreatedAt: now}
	if err := db.Create(&answer).Error; err == nil {
		t.Fatal("expected user_answers foreign-key violation")
	}
	choice := entity.QuestionChoice{ID: uuid.New(), QuestionID: uuid.New(), Code: "UNKNOWN_QUESTION_CHOICE", Label: "Unknown", SortOrder: 999, IsActive: true, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&choice).Error; err == nil {
		t.Fatal("expected question_choices.question_id foreign-key violation")
	}
}
