package telegram

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

// BotCommand структура команды бота
type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// BotCommandScope область действия меню команд
type BotCommandScope struct {
	Type string `json:"type"`
}

// SetMyCommandsRequest запрос на установку команд
type SetMyCommandsRequest struct {
	Commands []BotCommand     `json:"commands"`
	Scope    *BotCommandScope `json:"scope,omitempty"`
}

func InitializeBot(store *storage.Store) (*Bot, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используются переменные окружения")
	}

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")

	if token == "" {
		return nil, fmt.Errorf("переменная окружения TELEGRAM_BOT_TOKEN не задана")
	}
	if chatID == "" {
		return nil, fmt.Errorf("переменная окружения TELEGRAM_CHAT_ID не задана")
	}

	bot := createBot(token, chatID, store)

	return bot, nil
}

func createBot(token, chatID string, store *storage.Store) *Bot {
	bot := Bot{
		Token:  token,
		ChatID: chatID,
		Client: &http.Client{
			Timeout: 60 * time.Second,
		},
		BotURL: fmt.Sprintf("https://api.telegram.org/bot%s", token),
		Store:  store,
	}

	log.Println("Telegram бот инициализирован")

	if err := bot.setBotCommands(); err != nil {
		log.Printf("Ошибка установки меню команд: %v", err)
	}
	if err := bot.sendStartupNotification(); err != nil {
		log.Printf("Ошибка стартового уведомления: %v", err)
	}

	return &bot
}

func (bot *Bot) setBotCommands() error {
	commands := []BotCommand{
		{Command: "start", Description: "Начать работу"},
		{Command: "find", Description: "Поиск манги по названию"},
		{Command: "list", Description: "Мои подписки"},
		{Command: "help", Description: "Справка"},
	}

	request := SetMyCommandsRequest{
		Commands: commands,
		Scope:    nil,
	}

	if err := bot.callTelegram("setMyCommands", request, nil); err != nil {
		return fmt.Errorf("ошибка установки команд: %w", err)
	}

	log.Println("Меню команд бота установлено")
	return nil
}

func (bot *Bot) sendStartupNotification() error {
	chatID, err := strconv.ParseInt(bot.ChatID, 10, 64)
	if err != nil {
		return err
	}

	return bot.sendMessageToChat(chatID, textStartupNotification, false)
}

func (bot *Bot) StartPolling() {
	log.Println("Запуск Telegram polling...")

	var offset int64 = 0

	for {
		updates, err := bot.getUpdates(offset)
		if err != nil {
			log.Printf("Ошибка получения обновлений: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, update := range updates {
			offset = update.UpdateID + 1

			if update.CallbackQuery != nil {
				bot.handleCallbackQuery(update.CallbackQuery)
				continue
			}

			bot.handleChatMessage(update)
		}
	}
}
