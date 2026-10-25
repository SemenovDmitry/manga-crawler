package adminapi

import (
	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

type Store interface {
	// auth
	GetAdminSession(tokenHash string) (storage.AdminSession, error)
	RotateAdminSession(oldHash string, new storage.AdminSession) (bool, error)
	RevokeAdminSession(tokenHash string) error
	CreateAdminSession(sess storage.AdminSession) error

	// sources
	GetSources(filter storage.SourceFilter) ([]storage.Source, error)
	SetSourceEnabled(id int, enabled bool) (storage.Source, error)
	ResetSourceFullCrawl(id int) (storage.Source, error)

	// chat_users
	ListChatUsers(filter storage.ChatUserFilter) ([]storage.ChatUser, error)
	SetChatUserBlocked(chatID int64, blocked bool) (storage.ChatUser, error)

	// manga
	ListManga(filter storage.MangaFilter) ([]storage.Manga, error)
	GetManga(id int64) (storage.Manga, error)
	CreateManga(m storage.Manga) (storage.Manga, error)
	UpdateManga(id int64, patch storage.MangaPatch) (storage.Manga, error)
	DeleteManga(id int64) (storage.Manga, error)
}

// Проверка, что реальное хранилище всегда удовлетворяет интерфейсу.
var _ Store = (*storage.Store)(nil)
