package main

import (
	"context"
	"log"
	"os"
	"time"

	pgstore "github.com/lingmirror/image-service/internal/postgres"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	store, err := pgstore.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		log.Fatal(err)
	}
	log.Print("image service schema migration complete")
}
