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

// InitializeBot loads .env credentials and constructs the bot, installing
// command menu and sending a startup notification when enabled.
func InitializeBot() *Bot {
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используются переменные окружения")
	}

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")

	tgBot := createBot(token, chatID)

	return tgBot
}

// createBot builds the bot from credentials; without token or chatID it
// returns a disabled bot that silently skips all API calls.
func createBot(token, chatID string) *Bot {
	if token == "" || chatID == "" {
		log.Println("Telegram бот отключен: отсутствуют токен или chatID")
		return &Bot{Enabled: false}
	}

	tgBot := Bot{
		Token:  token,
		ChatID: chatID,
		Client: &http.Client{
			Timeout: 60 * time.Second,
		},
		BotURL:  fmt.Sprintf("https://api.telegram.org/bot%s", token),
		Enabled: true,
	}

	log.Println("Telegram бот инициализирован")

	tgBot.setBotCommands()
	tgBot.sendStartupNotification()

	return &tgBot
}

// setBotCommands registers the bot command menu via setMyCommands.
// A disabled bot is a no-op returning nil.
func (bot *Bot) setBotCommands() error {
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

	resp, err := bot.postJSON("setMyCommands", jsonData)
	if err != nil {
		return fmt.Errorf("ошибка установки команд: %w", err)
	}

	var apiResp APIResponse
	if err := decodeAPIResponse(resp, &apiResp); err != nil {
		return err
	}

	if !apiResp.OK {
		return fmt.Errorf("ошибка API: %s", apiResp.Description)
	}

	log.Println("Меню команд бота установлено")
	return nil
}

// StartPolling runs the blocking long polling loop, routing every message update
// to HandleChatMessage; a disabled bot returns immediately.
func (bot *Bot) StartPolling() {
	if !bot.Enabled {
		log.Println("Polling не запущен: бот отключен")
		return
	}

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
			bot.handleChatMessage(update)
		}
	}
}

// getUpdates fetches new updates via long polling (30s hold) restricted to message updates.
func (bot *Bot) getUpdates(offset int64) ([]Update, error) {
	requestURL := fmt.Sprintf("%s/getUpdates?offset=%d&timeout=30&allowed_updates=%s",
		bot.BotURL, offset, url.QueryEscape(`["message"]`))

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	resp, err := bot.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса: %w", err)
	}

	var response GetUpdatesResponse
	if err := decodeAPIResponse(resp, &response); err != nil {
		return nil, err
	}

	if !response.OK {
		return nil, fmt.Errorf("ошибка API Telegram: %s", response.Description)
	}

	return response.Result, nil
}

// postJSON sends a POST request with a JSON body to the given bot API method
// through the configured Doer and returns the response.
func (bot *Bot) postJSON(method string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/%s", bot.BotURL, method),
		bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return bot.Client.Do(req)
}

// decodeAPIResponse reads and closes the response body and unmarshals it into v.
func decodeAPIResponse(resp *http.Response, v any) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	return nil
}
