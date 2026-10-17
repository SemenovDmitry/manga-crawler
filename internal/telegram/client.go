package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// BotCommand структура команды бота
type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// SetMyCommandsRequest запрос на установку команд
type SetMyCommandsRequest struct {
	Commands []BotCommand `json:"commands"`
}

func InitializeTelegramBot() *TelegramBot {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка загрузки .env файла")
	}

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")

	tgBot := createTelegramBot(token, chatID)

	return tgBot
}

func createTelegramBot(token, chatID string) *TelegramBot {
	if token == "" || chatID == "" {
		log.Println("Telegram бот отключен: отсутствуют токен или chatID")
		return &TelegramBot{Enabled: false}
	}

	tgBot := TelegramBot{
		Token:  token,
		ChatID: chatID,
		Client: &http.Client{
			Timeout: 60 * time.Second,
		},
		BotURL:  fmt.Sprintf("https://api.telegram.org/bot%s", token),
		Enabled: true,
	}

	if tgBot.Enabled {
		log.Println("Telegram бот инициализирован")

		SetBotCommands(&tgBot)

		SendStartupNotification(&tgBot)
	} else {
		log.Println("Telegram бот отключен")
	}

	return &tgBot
}

func SetBotCommands(bot *TelegramBot) error {
	if !bot.Enabled {
		return nil
	}

	commands := SetMyCommandsRequest{
		Commands: []BotCommand{
			{Command: "start", Description: "Начать работу"},
			{Command: "help", Description: "Справка"},
		},
	}

	jsonData, err := json.Marshal(commands)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга команд: %w", err)
	}

	resp, err := bot.Client.Post(
		fmt.Sprintf("%s/setMyCommands", bot.BotURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("ошибка установки команд: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	if !apiResp.OK {
		return fmt.Errorf("ошибка API: %s", apiResp.Description)
	}

	log.Println("Меню команд бота установлено")
	return nil
}

// StartPolling runs the long polling loop, routing every update to HandleChatMessage.
func StartPolling(bot *TelegramBot) {
	if !bot.Enabled {
		log.Println("Polling не запущен: бот отключен")
		return
	}

	log.Println("Запуск Telegram polling...")

	var offset int64 = 0

	for {
		updates, err := getUpdates(bot, offset)
		if err != nil {
			log.Printf("Ошибка получения обновлений: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, update := range updates {
			offset = update.UpdateID + 1
			HandleChatMessage(bot, update)
		}
	}
}

// getUpdates fetches new updates via long polling (30s hold) restricted to message updates.
func getUpdates(bot *TelegramBot, offset int64) ([]Update, error) {
	url := fmt.Sprintf("%s/getUpdates?offset=%d&timeout=30&allowed_updates=%s",
		bot.BotURL, offset, url.QueryEscape(`["message"]`))

	resp, err := bot.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	var response GetUpdatesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	if !response.OK {
		return nil, fmt.Errorf("ошибка API Telegram: %s", response.Description)
	}

	return response.Result, nil
}
