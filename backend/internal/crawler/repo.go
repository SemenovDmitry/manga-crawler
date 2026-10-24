package crawler

import (
	"database/sql"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

// mangaRepo — операции с БД для одной карточки манги (processor.go)
// и очереди уведомлений (notifier.go).
type mangaRepo interface {
	GetMangaState(sourceID int, urlPart string) (storage.MangaState, bool, error)
	UpsertManga(
		sourceID int,
		urlPart, title, altTitles, latestChapter, publicationStatus string,
		chaptersCount int,
	) (int64, error)
	SubscriberChatIDs(mangaID int64) ([]int64, error)
	EnqueueNotification(chatID, mangaID int64, text string) error
}

// passRepo — операции с БД для проходов по источникам (crawler.go).
type passRepo interface {
	// GetEnabledSources — включённые источники (sources.enabled);
	// перечитывается в New и перед каждым инкрементальным проходом.
	GetEnabledSources() ([]storage.Source, error)
	// GetMangaURLs — все url-части манг источника (дифф sitemap/каталога).
	GetMangaURLs(sourceID int) ([]string, error)
	// GetStaleSubscribedManga — подписанные манги, не проверявшиеся дольше интервала.
	GetStaleSubscribedManga(sourceID, staleSeconds, limit int) ([]storage.Manga, error)
	// MarkSourceFullCrawled фиксирует завершённый полный обход и возвращает
	// назначенное время следующего.
	MarkSourceFullCrawled(id int) (sql.NullTime, error)
	// GetMangaCountBySource — статистика для логов.
	GetMangaCountBySource() (map[string]int, error)
}

// repo — всё, что краулер ждёт от хранилища. *storage.Store удовлетворяет
// ему неявно; в тестах подставляется мок.
type repo interface {
	mangaRepo
	passRepo
}

// Компиляторная проверка: реальное хранилище удовлетворяет контракту краулера.
var _ repo = (*storage.Store)(nil)
