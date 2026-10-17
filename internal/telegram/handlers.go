package telegram

import (
	"fmt"
	"strings"
)

// HandleChatMessage routes an incoming update's chat message to the matching command handler.
// Updates without a message (edited messages, callback queries, etc.) are ignored.
func HandleChatMessage(bot *TelegramBot, update Update) {
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
		handleStart(bot, chatID)
	case text == "/help":
		handleHelp(bot, chatID)
	default:
		SendMessageToChat(bot, fmt.Sprintf("%d", chatID), TextUnknownCommand, true)
	}
}

func handleStart(bot *TelegramBot, chatID int64) {
	sendMessageToUser(bot, chatID, TextStart)
}

func handleHelp(bot *TelegramBot, chatID int64) {
	sendMessageToUser(bot, chatID, TextHelp)
}
