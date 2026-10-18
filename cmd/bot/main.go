package main

import (
	"log"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/telegram"
)

// main initializes the Telegram bot, starts long polling in a goroutine and blocks forever.
func main() {

	db := storage.GetConnection()
	defer storage.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("База данных недоступна: %v", err)
	}

	if err := storage.RunMigrations(db); err != nil {
		log.Fatalf("Ошибка миграций: %v", err)
	}

	tgbot := telegram.InitializeBot()

	go tgbot.StartPolling()
	log.Println("Бот запущен, ожидание сообщений...")

	// Блокируем main: выход из main завершает все горутины, включая polling
	select {}
}
