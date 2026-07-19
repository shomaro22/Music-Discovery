package main

import (
	"log"
	"music-discovery/backend/db"
	"music-discovery/backend/migrate"
)

func main() {
	postgres, err := db.NewPostgres()
	if err != nil {
		log.Fatal(err)
	}
	if err := migrate.CreateUsers(postgres); err != nil {
		log.Fatal(err)
	}
	if err := migrate.CreateRefreshTokens(postgres); err != nil {
		log.Fatal(err)
	}
	log.Println("migration completed")
}
