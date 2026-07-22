package main

import (
	"context"
	"log"
)

func main() {
	if err := start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
