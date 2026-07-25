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
	return seedQuestions(db)
}

func seedQuestions(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, seed := range questionSeeds {
			question := entity.Question{Code: seed.Code, Title: seed.Title, AnswerType: seed.AnswerType, MinSelections: seed.MinSelections, MaxSelections: seed.MaxSelections, SortOrder: seed.SortOrder, IsActive: true}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "code"}}, DoUpdates: clause.AssignmentColumns([]string{"title", "answer_type", "min_selections", "max_selections", "sort_order", "is_active", "updated_at"})}).Create(&question).Error; err != nil {
				return fmt.Errorf("seed question %s: %w", seed.Code, err)
			}
			if err := tx.Where("code = ?", seed.Code).First(&question).Error; err != nil {
				return err
			}
			for _, item := range seed.Choices {
				choice := entity.QuestionChoice{QuestionID: question.ID, Code: item.Code, Label: item.Label, SortOrder: item.SortOrder, IsActive: true}
				if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "code"}}, DoUpdates: clause.AssignmentColumns([]string{"question_id", "label", "sort_order", "is_active", "updated_at"})}).Create(&choice).Error; err != nil {
					return fmt.Errorf("seed choice %s: %w", item.Code, err)
				}
			}
		}
		return nil
	})
}
