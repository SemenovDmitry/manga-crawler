package telegram

import (
	"errors"
	"log"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

func (bot *Bot) userBlocked(chatID int64) bool {
	if bot.Store == nil {
		return false
	}

	user, err := bot.Store.GetChatUser(chatID)
	switch {
	case errors.Is(err, storage.ErrChatUserNotFound):
		// Пользователя ещё нет в БД — он не заблокирован: бот открыт по умолчанию.
		return false
	case err != nil:
		// Реальная ошибка БД — fail-closed: не обслуживаем, пока БД недоступна.
		log.Printf("userBlocked: ошибка проверки чата %d: %v", chatID, err)
		return true
	}

	return user.Blocked || !user.IsActive
}
