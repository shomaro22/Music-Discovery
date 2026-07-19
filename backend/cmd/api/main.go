package main

import (
	"context"
	"log"

	"music-discovery/backend/server"
)

func main() {
	if err := server.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
