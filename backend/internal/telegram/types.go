package telegram

import (
	"net/http"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Bot struct {
	Token  string
	ChatID string
	Client Doer
	BotURL string
	Store  *storage.Store
}

type MessageRequest struct {
	ChatID                string                `json:"chat_id"`
	Text                  string                `json:"text"`
	ParseMode             string                `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool                  `json:"disable_web_page_preview,omitempty"`
	ReplyMarkup           *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

// EditMessageTextRequest запрос на изменение текста отправленного сообщения.
type EditMessageTextRequest struct {
	ChatID                string                `json:"chat_id"`
	MessageID             int64                 `json:"message_id"`
	Text                  string                `json:"text"`
	ParseMode             string                `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool                  `json:"disable_web_page_preview,omitempty"`
	ReplyMarkup           *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

// AnswerCallbackQueryRequest запрос на подтверждение нажатия inline-кнопки.
type AnswerCallbackQueryRequest struct {
	CallbackQueryID string `json:"callback_query_id"`
	Text            string `json:"text,omitempty"`
}

// InlineKeyboardButton кнопка inline-клавиатуры.
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

// InlineKeyboardMarkup inline-клавиатура (ряды кнопок).
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// APIResponse структура ответа от Telegram API
type APIResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

// GetUpdatesResponse ответ на getUpdates
type GetUpdatesResponse struct {
	OK          bool     `json:"ok"`
	Result      []Update `json:"result"`
	Description string   `json:"description,omitempty"`
}

// Update структура обновления от Telegram
type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

// CallbackQuery нажатие inline-кнопки.
type CallbackQuery struct {
	ID      string   `json:"id"`
	From    *User    `json:"from,omitempty"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data,omitempty"`
}

type Message struct {
	MessageID int64  `json:"message_id"`
	From      *User  `json:"from,omitempty"`
	Chat      *Chat  `json:"chat"`
	Date      int64  `json:"date"`
	Text      string `json:"text,omitempty"`
}

type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

type Chat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}
