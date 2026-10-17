package telegram

import "net/http"

// TelegramBot конфигурация Telegram бота
type TelegramBot struct {
	Token   string
	ChatID  string
	Client  *http.Client
	BotURL  string
	Enabled bool
}

// Message структура для отправки сообщения
type Message struct {
	ChatID                string `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview,omitempty"`
}

// APIResponse структура ответа от Telegram API
type APIResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

// Update структура обновления от Telegram
type Update struct {
	UpdateID int64      `json:"update_id"`
	Message  *TgMessage `json:"message,omitempty"`
}

// TgMessage структура сообщения от Telegram
type TgMessage struct {
	MessageID int64   `json:"message_id"`
	From      *TgUser `json:"from,omitempty"`
	Chat      *TgChat `json:"chat"`
	Date      int64   `json:"date"`
	Text      string  `json:"text,omitempty"`
}

// TgUser пользователь Telegram (из API)
type TgUser struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// TgChat чат Telegram
type TgChat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// GetUpdatesResponse ответ на getUpdates
type GetUpdatesResponse struct {
	OK     bool     `json:"ok"`
	Result []Update `json:"result"`
}
