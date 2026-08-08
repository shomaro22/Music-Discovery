package migrate

import (
	"fmt"
	"gorm.io/gorm"
	"music-discovery/backend/entity"
)

func CreateFeedbackTables(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&entity.Feedback{},
		&entity.UserArtistWeight{},
		&entity.UserSongWeight{},
		&entity.UserProfile{},
	); err != nil {
		return fmt.Errorf("migrate feedback tables: %w", err)
	}
	statements := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_feedback_recommendation_detail ON feedbacks (recommendation_detail_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_user_artist_weight ON user_artist_weights (user_id, artist_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_user_song_weight ON user_song_weights (user_id, song_id)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("create feedback index: %w", err)
		}
	}
	return nil
}
