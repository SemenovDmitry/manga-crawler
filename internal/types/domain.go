package types

import "time"

// SourceName тип парсера (enum)
type SourceName string

const (
	SourceReadmanga SourceName = "readmanga"
	SourceMintmanga SourceName = "mintmanga"
)

// AllSources список всех доступных парсеров
var AllSources = []SourceName{
	SourceReadmanga,
	SourceMintmanga,
}

// IsValid проверяет, является ли имя валидным парсером
func (s SourceName) IsValid() bool {
	switch s {
	case SourceReadmanga, SourceMintmanga:
		return true
	}
	return false
}

// Source источник/сайт для парсинга манги (хранится в БД)
type Source struct {
	ID         int        `db:"id" json:"id"`                   // Уникальный идентификатор источника
	ParserName SourceName `db:"parser_name" json:"parser_name"` // Имя парсера (readmanga, mintmanga)
	BaseURL    string     `db:"base_url" json:"base_url"`       // Базовый URL сайта (поддомен)
	IsActive   bool       `db:"is_active" json:"is_active"`     // Активен ли источник
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`   // Дата добавления
	UpdatedAt  time.Time  `db:"updated_at" json:"updated_at"`   // Дата последнего обновления
}

// Manga манга
type Manga struct {
	ID               int            `db:"id" json:"id"`                                 // Уникальный идентификатор манги
	SourceID         int            `db:"source_id" json:"source_id"`                   // ID источника (внешний ключ на sources)
	URL              string         `db:"url" json:"url"`                               // Полный URL страницы манги
	Title            string         `db:"title" json:"title"`                           // Название манги
	LastChapterURL   string         `db:"last_chapter_url" json:"last_chapter_url"`     // URL последней известной главы
	LastChapterTitle string         `db:"last_chapter_title" json:"last_chapter_title"` // Название последней главы
	LastCheckAt      *time.Time     `db:"last_check_at" json:"last_check_at"`           // Время последней проверки обновлений
	CreatedAt        time.Time      `db:"created_at" json:"created_at"`                 // Дата добавления
	UpdatedAt        time.Time      `db:"updated_at" json:"updated_at"`                 // Дата последнего обновления
	Chapters         []Chapter      `db:"-" json:"chapters,omitempty"`                  // Главы (не из БД, заполняется отдельно)
	Subscribers      []TelegramUser `db:"-" json:"subscribers,omitempty"`               // Подписчики (не из БД, заполняется отдельно)
}

// UserSubscription подписка пользователя на мангу
type UserSubscription struct {
	ID             int       `db:"id" json:"id"`                 // Уникальный идентификатор подписки
	TelegramUserID int64     `db:"user_id" json:"user_id"`       // ID пользователя (внешний ключ на telegram_users)
	MangaID        int       `db:"manga_id" json:"manga_id"`     // ID манги (внешний ключ на manga)
	Notify         bool      `db:"notify" json:"notify"`         // Отправлять ли уведомления о новых главах
	CreatedAt      time.Time `db:"created_at" json:"created_at"` // Дата подписки
}

// Chapter глава манги
type Chapter struct {
	ID           int       `db:"id" json:"id"`                       // Уникальный идентификатор главы
	MangaID      int       `db:"manga_id" json:"manga_id"`           // ID манги (внешний ключ на manga)
	URL          string    `db:"url" json:"url"`                     // URL главы
	Title        string    `db:"title" json:"title"`                 // Название главы
	DiscoveredAt time.Time `db:"discovered_at" json:"discovered_at"` // Дата обнаружения главы
}

// TelegramUser пользователь Telegram
type TelegramUser struct {
	ID        int64     `db:"id" json:"id"`                 // Уникальный идентификатор пользователя в Telegram (используется как ChatID)
	Username  string    `db:"username" json:"username"`     // Имя пользователя (@username)
	FirstName string    `db:"first_name" json:"first_name"` // Имя
	LastName  string    `db:"last_name" json:"last_name"`   // Фамилия
	IsActive  bool      `db:"is_active" json:"is_active"`   // Активен ли пользователь
	CreatedAt time.Time `db:"created_at" json:"created_at"` // Дата регистрации
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"` // Дата последнего обновления
}
