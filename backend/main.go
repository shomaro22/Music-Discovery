package main

import (
	"context"
	"log"

	"music-discovery/backend/db"
	"music-discovery/backend/migrate"
	"music-discovery/backend/server"
)

func main() {
	if err := runMigration(); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	if err := server.Start(context.Background()); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func runMigration() error {
	postgres, err := db.NewPostgres()
	if err != nil {
		return err
	}

	if err := migrate.CreateUsers(postgres); err != nil {
		return err
	}

	if err := migrate.CreateRefreshTokens(postgres); err != nil {
		return err
	}

	log.Println("migration completed")

	return nil
}
