package telegram

import (
	"strings"
)

// handleChatMessage routes an incoming update's chat message to the matching command handler.
// Updates without a message (edited messages, callback queries, etc.) are ignored.
func (bot *Bot) handleChatMessage(update Update) {
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
		bot.sendMessageToChat(chatID, textStart, true)
	case text == "/help":
		bot.sendMessageToChat(chatID, textHelp, true)
	default:
		bot.sendMessageToChat(chatID, textUnknownCommand, true)
	}
}
