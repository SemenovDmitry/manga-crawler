package telegram

import (
	"strings"
)

// HandleChatMessage routes an incoming update's chat message to the matching command handler.
// Updates without a message (edited messages, callback queries, etc.) are ignored.
func (bot *Bot) HandleChatMessage(update Update) {
	if update.Message == nil {
		return
	}

	msg := update.Message
	if msg.From == nil {
		return
	}

	text := strings.TrimSpace(msg.Text)
	chatID := msg.Chat.ID

	switch {
	case text == "/start":
		bot.handleStart(chatID)
	case text == "/help":
		bot.handleHelp(chatID)
	default:
		bot.SendMessageToChat(chatID, TextUnknownCommand, true)
	}
}

// handleStart replies to the chat with the greeting text.
func (bot *Bot) handleStart(chatID int64) {
	bot.SendMessageToChat(chatID, TextStart, true)
}

// handleHelp replies to the chat with the help text.
func (bot *Bot) handleHelp(chatID int64) {
	bot.SendMessageToChat(chatID, TextHelp, true)
}
