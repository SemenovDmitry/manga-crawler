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

	"github.com/SemenovDmitry/manga-crawler-backend/internal/adminapi"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

func main() {
	_ = godotenv.Load()

	cfg := adminapi.ConfigFromEnv()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Конфигурация админки: %v", err)
	}

	store, err := storage.NewStore()
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer store.Close()

	// Housekeeping: истёкшие refresh-сессии при старте не нужны.
	if n, err := store.DeleteExpiredAdminSessions(); err != nil {
		log.Printf("Предупреждение: очистка истёкших сессий: %v", err)
	} else if n > 0 {
		log.Printf("Удалено истёкших сессий: %d", n)
	}

	srv := adminapi.New(store, cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		log.Printf("Admin API запущен на %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Ошибка HTTP-сервера: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Остановка Admin API...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Принудительная остановка: %v", err)
	}
}
