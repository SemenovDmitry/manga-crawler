package telegram

import (
	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

// Backend — всё, что бот ждёт от хранилища: узкий consumer-defined
// interface вместо полной поверхности storage.Store (38 методов). Семантика
// подписок живёт в storage/subscriptions.go, очереди — storage/notifications.go:
// интерфейс лишь перечисляет, что нужно handler-логике бота.
// *storage.Store удовлетворяет ему неявно; в тестах — хранилище в памяти
// (второй adapter — потому seam настоящий).
type Backend interface {
	// Пользователи чатов (storage/users.go): регистрация по /start,
	// проверка доступа, авто-снятие при блокировке бота пользователем.
	CreateChatUser(user storage.ChatUser) (storage.ChatUser, error)
	GetChatUser(chatID int64) (storage.ChatUser, error)
	MarkChatUserUnavailable(chatID int64) error

	// Манга и подписки (storage/subscriptions.go): поиск, карточка,
	// идемпотентная подписка/отписка, список подписок чата.
	SearchManga(query string, limit int) ([]storage.Manga, error)
	GetManga(id int64) (storage.Manga, error)
	Subscribe(chatID, mangaID int64) (bool, error)
	Unsubscribe(chatID, mangaID int64) (bool, error)
	UserSubscriptions(chatID int64) ([]storage.Manga, error)

	// Очередь уведомлений (storage/notifications.go): заявка порции
	// неотправленных и отметка доставленных.
	PendingNotifications(limit int) ([]storage.Notification, error)
	MarkNotificationsSent(ids []int64) error
}

// Компиляторная проверка: реальное хранилище удовлетворяет контракту бота.
var _ Backend = (*storage.Store)(nil)
