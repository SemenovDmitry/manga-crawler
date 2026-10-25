package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/joho/godotenv"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

func main() {
	_ = godotenv.Load()

	store, err := storage.NewStore()
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer store.Close()

	cfg := crawler.ConfigFromEnv()

	c := crawler.New(store, cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	c.Run(ctx)
}
