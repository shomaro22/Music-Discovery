package migrate

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"music-discovery/backend/entity"
)

type tagSeed struct {
	Name        string
	Slug        string
	Category    string
	Description string
}

var tagCategoryByQuestionCode = map[string]string{
	entity.QuestionCodeMood:       "mood",
	entity.QuestionCodeElement:    "feature",
	entity.QuestionCodeAtmosphere: "scene",
	entity.QuestionCodeGenre:      "genre",
}

func systemTagSeeds() []tagSeed {
	result := make([]tagSeed, 0, 26)
	for _, question := range questionSeeds {
		category, ok := tagCategoryByQuestionCode[question.Code]
		if !ok {
			continue
		}
		for _, choice := range question.Choices {
			result = append(result, tagSeed{
				Name:        choice.Label,
				Slug:        strings.ToLower(strings.ReplaceAll(choice.Code, "_", "-")),
				Category:    category,
				Description: fmt.Sprintf("%sの回答と推薦スコアで使用するシステムTag", question.Title),
			})
		}
	}
	return result
}

// SeedTags creates the master Tags used by both the questionnaire and the
// recommendation algorithm. Upserts are keyed by Slug, so rerunning startup
// migrations is safe and does not create duplicates.
func SeedTags(db *gorm.DB) error {
	if db == nil {
		return errors.New("database is required")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, seed := range systemTagSeeds() {
			if err := upsertSystemTag(tx, seed); err != nil {
				return fmt.Errorf("upsert tag %s: %w", seed.Slug, err)
			}
		}
		return nil
	})
}

func upsertSystemTag(tx *gorm.DB, seed tagSeed) error {
	var matches []entity.Tag
	if err := tx.Unscoped().
		Where("slug = ? OR name = ?", seed.Slug, seed.Name).
		Find(&matches).Error; err != nil {
		return err
	}
	if len(matches) > 1 {
		return fmt.Errorf("multiple existing Tags match name=%s or slug=%s", seed.Name, seed.Slug)
	}
	if len(matches) == 1 {
		return tx.Unscoped().Model(&matches[0]).Updates(map[string]interface{}{
			"name": seed.Name, "slug": seed.Slug, "category": seed.Category,
			"description": seed.Description, "deleted_at": nil,
		}).Error
	}

	value := entity.Tag{
		Name:        seed.Name,
		Slug:        seed.Slug,
		Category:    seed.Category,
		Description: seed.Description,
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "slug"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"name": seed.Name, "category": seed.Category,
			"description": seed.Description, "deleted_at": nil,
		}),
	}).Create(&value).Error
}
