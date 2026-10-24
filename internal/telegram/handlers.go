package telegram

import (
	"log"
	"strconv"
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
	case text == "/find" || strings.HasPrefix(text, "/find "):
		bot.handleFind(chatID, strings.TrimPrefix(text, "/find"))
	case text == "/list":
		bot.handleList(chatID)
	default:
		bot.sendMessageToChat(chatID, textUnknownCommand, true)
	}
}

// handleCallbackQuery обрабатывает нажатия inline-кнопок.
func (bot *Bot) handleCallbackQuery(callback *CallbackQuery) {
	if callback.Message == nil || callback.Message.Chat == nil {
		bot.answerCallbackQuery(callback.ID, textButtonExpired)
		return
	}

	chatID := callback.Message.Chat.ID
	messageID := callback.Message.MessageID

	prefix, arg, found := strings.Cut(callback.Data, ":")
	if !found {
		bot.answerCallbackQuery(callback.ID, textUnknownAction)
		return
	}

	mangaID, err := strconv.ParseInt(strings.TrimSpace(arg), 10, 64)
	if err != nil {
		bot.answerCallbackQuery(callback.ID, textUnknownAction)
		return
	}

	switch prefix {
	case callbackFollow:
		reply := bot.subscribeManga(chatID, mangaID)
		bot.confirmCallback(callback, chatID, messageID, reply)
	case callbackUnfollow:
		reply := bot.unsubscribeManga(chatID, mangaID)
		bot.confirmCallback(callback, chatID, messageID, reply)
	default:
		bot.answerCallbackQuery(callback.ID, textUnknownAction)
	}
}

// confirmCallback отвечает на нажатие кнопки и обновляет сообщение подтверждением.
// Если сообщение отредактировать не удалось (например, оно старше 48 часов),
// подтверждение уходит отдельным сообщением.
func (bot *Bot) confirmCallback(callback *CallbackQuery, chatID, messageID int64, reply string) {
	if err := bot.editMessageText(chatID, messageID, reply, true, nil); err != nil {
		log.Printf("Не удалось изменить сообщение %d в чате %d: %v", messageID, chatID, err)
		bot.sendMessageToChat(chatID, reply, true)
	}

	bot.answerCallbackQuery(callback.ID, "")
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
