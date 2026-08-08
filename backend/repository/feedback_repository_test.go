package repository

import (
	"context"
	"strings"
	"testing"

	"music-discovery/backend/entity"
	"music-discovery/backend/migrate"

	"github.com/google/uuid"
)

func normalizeQuestionChoiceCode(code string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(code), "_", "-"))
}

func TestFeedbackTagSlugMatchesQuestionChoiceCode(t *testing.T) {
	tests := []struct {
		code string
		slug string
	}{
		{code: "GENRE_ROCK", slug: "genre-rock"},
		{code: "ELEMENT_LYRICS", slug: "element-lyrics"},
		{code: "ATMOSPHERE_BRIGHT", slug: "atmosphere-bright"},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			if got := normalizeQuestionChoiceCode(tt.code); got != tt.slug {
				t.Fatalf("normalized code=%q want=%q", got, tt.slug)
			}
		})
	}
}

func TestFeedbackRepositoryTagJoinUsesNormalizedCodePostgres(t *testing.T) {
	db := openTestPostgres(t)
	if err := migrate.Run(db, migrate.Config{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("TRUNCATE TABLE question_choices, questions CASCADE").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}

	question := entity.Question{
		ID: uuid.New(), Code: entity.QuestionCodeGenre, Title: "genre",
		AnswerType: entity.QuestionAnswerTypeSingle, MinSelections: 1, SortOrder: 1, IsActive: true,
	}
	if err := db.Create(&question).Error; err != nil {
		t.Fatalf("create question: %v", err)
	}
	choices := []entity.QuestionChoice{
		{ID: uuid.New(), QuestionID: question.ID, Code: "GENRE_ROCK", Label: "Rock", SortOrder: 1, IsActive: true},
		{ID: uuid.New(), QuestionID: question.ID, Code: "ELEMENT_LYRICS", Label: "Lyrics", SortOrder: 2, IsActive: true},
		{ID: uuid.New(), QuestionID: question.ID, Code: "ATMOSPHERE_BRIGHT", Label: "Bright", SortOrder: 3, IsActive: true},
	}
	if err := db.Create(&choices).Error; err != nil {
		t.Fatalf("create choices: %v", err)
	}

	for _, slug := range []string{"genre-rock", "element-lyrics", "atmosphere-bright"} {
		var count int64
		err := db.Raw(`SELECT COUNT(*) FROM question_choices qc
            WHERE LOWER(REPLACE(qc.code, '_', '-')) = LOWER(?) AND qc.is_active = TRUE`, slug).Scan(&count).Error
		if err != nil {
			t.Fatalf("join %s: %v", slug, err)
		}
		if count != 1 {
			t.Fatalf("slug=%s matched=%d want=1", slug, count)
		}
	}

	_ = context.Background()
}
