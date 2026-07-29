package migrate

import (
	"fmt"

	"music-discovery/backend/entity"

	"gorm.io/gorm"
)

func CreateRecommendationTables(db *gorm.DB) error {
	// AutoMigrateより先にrecommendation_typeへ変更し、新旧両方の列が作られるのを防ぐ。
	if err := renameLegacyRecommendationTypeColumn(db); err != nil {
		return err
	}

	if err := db.AutoMigrate(
		&entity.Recommendation{},
		&entity.RecommendationDetail{},
	); err != nil {
		return fmt.Errorf("migrate recommendation tables: %w", err)
	}

	if err := createRecommendationUserForeignKey(db); err != nil {
		return err
	}

	if err := db.Exec(`
		DROP INDEX IF EXISTS uq_recommendation_position
	`).Error; err != nil {
		return fmt.Errorf(
			"drop legacy recommendation position index: %w",
			err,
		)
	}

	statements := []string{
		`
			CREATE UNIQUE INDEX IF NOT EXISTS uq_recommendation_type
			ON recommendation_details (
				recommendation_id,
				recommendation_type
			)
		`,
		`
			CREATE UNIQUE INDEX IF NOT EXISTS uq_recommendation_position
			ON recommendation_details (
				recommendation_id,
				position
			)
		`,
		`
			CREATE UNIQUE INDEX IF NOT EXISTS uq_recommendation_song
			ON recommendation_details (
				recommendation_id,
				song_id
			)
		`,
	}

	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf(
				"create recommendation unique index: %w",
				err,
			)
		}
	}

	return nil
}

func renameLegacyRecommendationTypeColumn(db *gorm.DB) error {
	const statement = `
		DO $$
		BEGIN
			IF EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = 'public'
				  AND table_name = 'recommendation_details'
				  AND column_name = 'type'
			)
			AND NOT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = 'public'
				  AND table_name = 'recommendation_details'
				  AND column_name = 'recommendation_type'
			) THEN
				ALTER TABLE recommendation_details
				RENAME COLUMN type TO recommendation_type;
			END IF;
		END
		$$;
	`

	if err := db.Exec(statement).Error; err != nil {
		return fmt.Errorf(
			"rename legacy recommendation type column: %w",
			err,
		)
	}

	return nil
}

func createRecommendationUserForeignKey(db *gorm.DB) error {
	const statement = `
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1
				FROM pg_constraint
				WHERE conname = 'fk_recommendations_user'
				  AND conrelid = 'recommendations'::regclass
			) THEN
				ALTER TABLE recommendations
				ADD CONSTRAINT fk_recommendations_user
				FOREIGN KEY (user_id)
				REFERENCES users(id)
				ON UPDATE CASCADE
				ON DELETE RESTRICT;
			END IF;
		END
		$$;
	`

	if err := db.Exec(statement).Error; err != nil {
		return fmt.Errorf(
			"create recommendation user foreign key: %w",
			err,
		)
	}

	return nil
}
