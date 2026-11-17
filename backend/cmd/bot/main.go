package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/joho/godotenv"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/telegram"
)

func main() {
	_ = godotenv.Load()

	cfg := telegram.ConfigFromEnv()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Конфигурация Telegram-бота: %v", err)
	}

	store, err := storage.NewStore()
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer store.Close()

	telegramBot, err := telegram.InitializeBot(cfg, store)
	if err != nil {
		log.Fatalf("Ошибка инициализации Telegram-бота: %v", err)
	}
	// store удовлетворяет telegram.Backend неявно: бот видит только
	// свои 10 операций хранилища, а не всю поверхность storage.

	// Завершение по Ctrl+C (SIGTERM из docker stop — по таймауту):
	// горутины останавливаются по контексту, недоставленные уведомления
	// остаются в очереди до следующего запуска.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go telegramBot.StartPolling(ctx)
	go telegramBot.StartNotificationDispatch(ctx)
	log.Println("Бот запущен, ожидание сообщений...")

	<-ctx.Done()
	log.Println("Остановка бота...")
}
