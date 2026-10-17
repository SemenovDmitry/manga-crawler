package main

import (
	"log"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/telegram"
)

// main initializes the Telegram bot, starts long polling in a goroutine and blocks forever.
func main() {

	tgbot := telegram.InitializeTelegramBot()

	// Запускаем polling для получения команд от пользователей
	go tgbot.StartPolling()
	log.Println("Бот запущен, ожидание сообщений...")

	// Блокируем main: выход из main завершает все горутины, включая polling
	select {}
}
