package migrate

import (
	"fmt"

	"music-discovery/backend/entity"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type questionSeed struct {
	Code          string
	Title         string
	AnswerType    entity.QuestionAnswerType
	MinSelections int
	MaxSelections *int
	SortOrder     int
	Choices       []choiceSeed
}

type choiceSeed struct {
	Code, Label string
	SortOrder   int
}

func intPointer(value int) *int { return &value }

var questionSeeds = []questionSeed{
	{entity.QuestionCodeMood, "今日の気分", entity.QuestionAnswerTypeSingle, 1, intPointer(1), 1, []choiceSeed{{"MOOD_UPLIFTING", "気分を上げたい", 1}, {"MOOD_CALM", "落ち着きたい", 2}, {"MOOD_FOCUS", "集中したい", 3}, {"MOOD_SAD", "切ない気分", 4}, {"MOOD_EXTRAORDINARY", "非日常を感じたい", 5}}},
	{entity.QuestionCodeElement, "重視する要素", entity.QuestionAnswerTypeMultiple, 1, nil, 2, []choiceSeed{{"ELEMENT_LYRICS", "歌詞", 1}, {"ELEMENT_VOCAL", "歌声", 2}, {"ELEMENT_MELODY", "メロディ", 3}, {"ELEMENT_WORLDVIEW", "世界観", 4}, {"ELEMENT_RHYTHM", "リズム", 5}}},
	{entity.QuestionCodeAtmosphere, "雰囲気", entity.QuestionAnswerTypeMultiple, 1, nil, 3, []choiceSeed{{"ATMOSPHERE_BRIGHT", "明るい", 1}, {"ATMOSPHERE_NIGHT", "夜っぽい", 2}, {"ATMOSPHERE_CALM", "落ち着いた", 3}, {"ATMOSPHERE_DEEP", "ディープ", 4}, {"ATMOSPHERE_CINEMATIC", "映画的", 5}, {"ATMOSPHERE_REFRESHING", "爽やか", 6}}},
	{entity.QuestionCodeGenre, "ジャンル", entity.QuestionAnswerTypeMultiple, 1, intPointer(3), 4, []choiceSeed{{"GENRE_POP", "ポップ", 1}, {"GENRE_ROCK", "ロック", 2}, {"GENRE_ALTERNATIVE", "オルタナティブ", 3}, {"GENRE_METAL", "メタル", 4}, {"GENRE_HIP_HOP", "ヒップホップ", 5}, {"GENRE_JAZZ", "ジャズ", 6}, {"GENRE_ELECTRONIC", "エレクトロニック", 7}, {"GENRE_CLASSICAL", "クラシック", 8}, {"GENRE_ANIME", "アニメ", 9}, {"GENRE_K_POP", "K-POP", 10}}},
	{entity.QuestionCodeDiscoveryLevel, "探索度", entity.QuestionAnswerTypeSingle, 1, intPointer(1), 5, []choiceSeed{{"DISCOVERY_LOW", "似た音楽が安心", 1}, {"DISCOVERY_MEDIUM", "少し新しい音楽", 2}, {"DISCOVERY_HIGH", "全く違う音楽にも挑戦", 3}}},
}

func CreateQuestionTables(db *gorm.DB) error {
	if err := db.AutoMigrate(&entity.Question{}, &entity.QuestionChoice{}, &entity.QuestionAnswerSession{}, &entity.UserAnswer{}, &entity.UserAnswerChoice{}, &entity.UserProfile{}, &entity.UserGenreWeight{}, &entity.UserElementWeight{}, &entity.UserAtmosphereWeight{}); err != nil {
		return err
	}
	if err := createUserProfileUserForeignKey(db); err != nil {
		return err
	}
	// This guarded backfill intentionally runs at startup because this project does not use
	// versioned migration files. Its WHERE clause makes repeated execution idempotent.
	if err := migrateExistingQuestionWeights(db); err != nil {
		return err
	}
	return seedQuestions(db)
}

func migrateExistingQuestionWeights(db *gorm.DB) error {
	tables := []string{"user_genre_weights", "user_element_weights", "user_atmosphere_weights"}
	for _, table := range tables {
		query := fmt.Sprintf(`UPDATE %s SET question_weight = weight WHERE question_weight = 0 AND feedback_weight = 0 AND weight <> 0`, table)
		if err := db.Exec(query).Error; err != nil {
			return fmt.Errorf("migrate existing question weight for %s: %w", table, err)
		}
	}
	return nil
}

func seedQuestions(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		activeQuestionCodes := make([]string, 0, len(questionSeeds))
		for _, seed := range questionSeeds {
			activeQuestionCodes = append(activeQuestionCodes, seed.Code)
			question := entity.Question{Code: seed.Code, Title: seed.Title, AnswerType: seed.AnswerType, MinSelections: seed.MinSelections, MaxSelections: seed.MaxSelections, SortOrder: seed.SortOrder, IsActive: true}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "code"}}, DoUpdates: clause.AssignmentColumns([]string{"title", "answer_type", "min_selections", "max_selections", "sort_order", "updated_at"})}).Create(&question).Error; err != nil {
				return fmt.Errorf("seed question %s: %w", seed.Code, err)
			}
			if err := tx.Where("code = ?", seed.Code).First(&question).Error; err != nil {
				return err
			}
			activeCodes := make([]string, 0, len(seed.Choices))
			for _, item := range seed.Choices {
				activeCodes = append(activeCodes, item.Code)
				choice := entity.QuestionChoice{QuestionID: question.ID, Code: item.Code, Label: item.Label, SortOrder: item.SortOrder, IsActive: true}
				if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "code"}}, DoUpdates: clause.AssignmentColumns([]string{"question_id", "label", "sort_order", "updated_at"})}).Create(&choice).Error; err != nil {
					return fmt.Errorf("seed choice %s: %w", item.Code, err)
				}
			}
			if err := tx.Model(&entity.QuestionChoice{}).Where("question_id = ? AND code NOT IN ?", question.ID, activeCodes).Update("is_active", false).Error; err != nil {
				return fmt.Errorf("deactivate removed choices for %s: %w", seed.Code, err)
			}
		}
		if err := tx.Model(&entity.Question{}).Where("code NOT IN ?", activeQuestionCodes).Update("is_active", false).Error; err != nil {
			return fmt.Errorf("deactivate removed questions: %w", err)
		}
		return nil
	})
}

func createUserProfileUserForeignKey(db *gorm.DB) error {
	const statement = `
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1
				FROM pg_constraint
				WHERE conname = 'fk_user_profiles_user'
				  AND conrelid = 'user_profiles'::regclass
			) THEN
				ALTER TABLE user_profiles
				ADD CONSTRAINT fk_user_profiles_user
				FOREIGN KEY (user_id)
				REFERENCES users(id)
				ON UPDATE CASCADE
				ON DELETE RESTRICT;
			END IF;
		END
		$$;
	`
	if err := db.Exec(statement).Error; err != nil {
		return fmt.Errorf("create user_profiles user foreign key: %w", err)
	}
	return nil
}
