package miniapi

import (
	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

type Store interface {
	// chat_users
	CreateChatUser(user storage.ChatUser) (storage.ChatUser, error)

	// manga
	SearchManga(query string, limit int) ([]storage.Manga, error)
	GetManga(id int64) (storage.Manga, error)

	// subscriptions
	IsSubscribed(chatID, mangaID int64) (bool, error)
	Subscribe(chatID, mangaID int64) (bool, error)
	Unsubscribe(chatID, mangaID int64) (bool, error)
	UserSubscriptions(chatID int64) ([]storage.Manga, error)
}

// Проверка, что реальное хранилище всегда удовлетворяет интерфейсу.
var _ Store = (*storage.Store)(nil)
