package migrate

import (
	"fmt"

	"gorm.io/gorm"
	"music-discovery/backend/entity"
)

func CreateRecommendationTables(db *gorm.DB) error {
	if err := db.AutoMigrate(&entity.Recommendation{}, &entity.RecommendationDetail{}); err != nil {
		return fmt.Errorf("migrate recommendation tables: %w", err)
	}
	// GORM creates uniqueIndex tags as PostgreSQL indexes. Drop the former
	// three-column index before recreating the position-only index with the same name.
	if err := db.Exec(`DROP INDEX IF EXISTS uq_recommendation_position`).Error; err != nil {
		return fmt.Errorf("drop legacy recommendation index: %w", err)
	}
	statements := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_recommendation_type ON recommendation_details (recommendation_id, recommendation_type)`,
		`CREATE UNIQUE INDEX uq_recommendation_position ON recommendation_details (recommendation_id, position)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_recommendation_song ON recommendation_details (recommendation_id, song_id)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("create recommendation unique index: %w", err)
		}
	}
	return nil
}
