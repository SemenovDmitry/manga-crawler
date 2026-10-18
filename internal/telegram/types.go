package telegram

import (
	"net/http"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

// Doer abstracts the HTTP transport used by the bot; *http.Client satisfies it directly.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Bot is the Telegram bot configuration.
type Bot struct {
	Token   string
	ChatID  string
	Client  Doer
	BotURL  string
	Enabled bool
	Store   *storage.Store
}

// MessageRequest is the outgoing sendMessage payload.
type MessageRequest struct {
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

// GetUpdatesResponse ответ на getUpdates
type GetUpdatesResponse struct {
	OK          bool     `json:"ok"`
	Result      []Update `json:"result"`
	Description string   `json:"description,omitempty"`
}

// Update структура обновления от Telegram
type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message,omitempty"`
}

// Message is a message received from Telegram.
type Message struct {
	MessageID int64  `json:"message_id"`
	From      *User  `json:"from,omitempty"`
	Chat      *Chat  `json:"chat"`
	Date      int64  `json:"date"`
	Text      string `json:"text,omitempty"`
}

// User is a Telegram user as returned by the API.
type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// Chat is a Telegram chat.
type Chat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}
