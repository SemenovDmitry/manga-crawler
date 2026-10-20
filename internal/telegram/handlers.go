package telegram

import (
	"log"
	"strings"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

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
		bot.saveChatUser(msg.From, chatID)
		bot.sendMessageToChat(chatID, textStart, true)
	case text == "/help":
		bot.sendMessageToChat(chatID, textHelp, true)
	default:
		bot.sendMessageToChat(chatID, textUnknownCommand, true)
	}
}

func (bot *Bot) saveChatUser(user *User, chatID int64) {
	if user == nil || bot.Store == nil {
		return
	}

	if _, err := bot.Store.CreateChatUser(storage.ChatUser{
		ChatID:   chatID,
		Username: user.Username,
		IsActive: true,
	}); err != nil {
		log.Printf("Ошибка сохранения пользователя %d: %v", user.ID, err)
	}
}
