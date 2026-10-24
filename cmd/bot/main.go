package main

import (
	"log"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/telegram"
)

func main() {
	store, err := storage.NewStore()
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer store.Close()

	telegramBot, err := telegram.InitializeBot(store)
	if err != nil {
		log.Fatalf("Ошибка инициализации Telegram-бота: %v", err)
	}

	go telegramBot.StartPolling()
	go telegramBot.StartNotificationDispatch()
	log.Println("Бот запущен, ожидание сообщений...")

	// Блокируем main: выход из main завершает все горутины, включая polling
	select {}
}
