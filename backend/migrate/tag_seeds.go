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

var curatedSubgenreTagSeeds = []tagSeed{
	{Name: "J-POP（サブジャンル）", Slug: "subgenre-j-pop", Category: "subgenre", Description: "日本向け初期カタログのJ-POP分類"},
	{Name: "J-ROCK（サブジャンル）", Slug: "subgenre-j-rock", Category: "subgenre", Description: "日本向け初期カタログのJ-ROCK分類"},
	{Name: "Alternative Rock（サブジャンル）", Slug: "subgenre-alternative-rock", Category: "subgenre", Description: "日本向け初期カタログのAlternative Rock分類"},
	{Name: "Melodic Punk（サブジャンル）", Slug: "subgenre-melodic-punk", Category: "subgenre", Description: "日本向け初期カタログのメロディックパンク分類"},
	{Name: "Punk Rock（サブジャンル）", Slug: "subgenre-punk-rock", Category: "subgenre", Description: "日本向け初期カタログのパンクロック分類"},
	{Name: "Post Hardcore（サブジャンル）", Slug: "subgenre-post-hardcore", Category: "subgenre", Description: "日本向け初期カタログのポストハードコア分類"},
	{Name: "Metalcore（サブジャンル）", Slug: "subgenre-metalcore", Category: "subgenre", Description: "日本向け初期カタログのメタルコア分類"},
	{Name: "Heavy Metal（サブジャンル）", Slug: "subgenre-heavy-metal", Category: "subgenre", Description: "日本向け初期カタログのヘヴィメタル分類"},
	{Name: "International Pop（サブジャンル）", Slug: "subgenre-international-pop", Category: "subgenre", Description: "日本で人気のある海外ポップ分類"},
	{Name: "K-POP（サブジャンル）", Slug: "subgenre-k-pop", Category: "subgenre", Description: "日本向け初期カタログのK-POP分類"},
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

func allTagSeeds() []tagSeed {
	profileTags := systemTagSeeds()
	result := make([]tagSeed, 0, len(profileTags)+len(curatedSubgenreTagSeeds))
	result = append(result, profileTags...)
	result = append(result, curatedSubgenreTagSeeds...)
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
		for _, seed := range allTagSeeds() {
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
