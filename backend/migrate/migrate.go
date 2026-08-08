package migrate

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// Config controls optional data seeding after schema migration.
// Schema migration itself always runs when Run is called.
type Config struct {
	AppEnvironment         string
	SeedDevelopmentUsers   bool
	SeedDevelopmentCatalog bool
}

type migrationStep struct {
	name string
	run  func(*gorm.DB) error
}

// Run applies every schema migration in dependency order and then executes
// explicitly enabled development seeds.
func Run(db *gorm.DB, config Config) error {
	if db == nil {
		return errors.New("database is required")
	}

	if err := runSchemaMigrations(db); err != nil {
		return err
	}
	if err := runDevelopmentSeeds(db, config); err != nil {
		return err
	}
	return nil
}

func runSchemaMigrations(db *gorm.DB) error {
	steps := []migrationStep{
		{name: "users", run: CreateUsers},
		{name: "refresh tokens", run: CreateRefreshTokens},
		{name: "password reset tokens", run: CreatePasswordResetTokens},
		{name: "admin tables", run: CreateAdminTables},
		{name: "question tables", run: CreateQuestionTables},
		{name: "recommendation tables", run: CreateRecommendationTables},
		{name: "feedback tables", run: CreateFeedbackTables},
	}

	for _, step := range steps {
		if err := step.run(db); err != nil {
			return fmt.Errorf("migrate %s: %w", step.name, err)
		}
	}
	return nil
}

func runDevelopmentSeeds(db *gorm.DB, config Config) error {
	if config.SeedDevelopmentUsers {
		if err := SeedDevelopmentUsersForEnvironment(db, config.AppEnvironment); err != nil {
			return fmt.Errorf("seed development users: %w", err)
		}
	}
	if config.SeedDevelopmentCatalog {
		if err := SeedDevelopmentCatalogForEnvironment(db, config.AppEnvironment); err != nil {
			return fmt.Errorf("seed development catalog: %w", err)
		}
	}
	return nil
}
