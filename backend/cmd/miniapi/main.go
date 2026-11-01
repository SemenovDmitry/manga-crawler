package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/miniapi"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

func main() {
	_ = godotenv.Load()

	cfg := miniapi.ConfigFromEnv()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Конфигурация miniapi: %v", err)
	}

	store, err := storage.NewStore()
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer store.Close()

	srv := miniapi.New(store, cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		log.Printf("Mini API запущен на %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Ошибка HTTP-сервера: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Остановка Mini API...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Принудительная остановка: %v", err)
	}
}
