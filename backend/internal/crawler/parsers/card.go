// Package parsers — контракты и общие типы драйверов источников.
//
// Каждый сайт (или семейство сайтов на одном движке) реализует свой
// драйвер в подпакете (grouple, mgeko); краулер, расписание и уведомления
// от движка не зависят. Общие для всех драйверов сущности — карточка
// манги, глава, статусы публикации — собраны здесь, чтобы подпакеты
// зависели только от parsers, а друг от друга — нет.
package parsers

import (
	"context"
	"errors"
)

// PageLoader — функция загрузки страницы, которую краулер передаёт
// драйверам с догрузкой данных (см. ChapterLoader): обёртка над Fetcher
// с паузой вежливости перед каждым запросом.
type PageLoader func(ctx context.Context, rawURL string) ([]byte, error)

// ChapterLoader — необязательная способность драйвера: догружать главы,
// которых нет в HTML карточки. Нужен источникам, где SSR отдаёт только
// счётчик глав, а список живёт в отдельном API (remanga). Краулер вызывает
// LoadChapters сразу после ParseMangaCard; ошибка догрузки не отменяет
// сохранение карточки — заголовок и статус уже распарсены.
type ChapterLoader interface {
	// LoadChapters дополняет card.Chapters. urlPart — url-часть карточки
	// ("manga/slug"), из неё драйвер строит адреса API. load загружает
	// страницы (краулер оборачивает в неё свои паузы и ретраи).
	LoadChapters(ctx context.Context, load PageLoader, urlPart string, card *MangaCard) error
}

// CardUserAgent — необязательная способность драйвера: свой User-Agent
// для загрузки карточек. Нужен источникам, которые отдают полный SSR
// только известным роботам (mangalib: браузерный UA получает пустой
// SPA-каркас, Googlebot-UA — полноценную страницу). Возвращается
// User-Agent целиком; краулер подставит его в запросы карточек только
// этого источника, остальные источники работают со стандартным UA.
type CardUserAgent interface {
	CardUserAgent() string
}

// Chapter — глава из списка на странице манги.
type Chapter struct {
	Number string
	Volume string
	Title  string
}

// MangaCard — данные, извлечённые из страницы манги.
type MangaCard struct {
	Title     string
	AltTitles string
	// PublicationStatus — статус публикации: ongoing, completed, hiatus,
	// cancelled, discontinued. Пустая строка — не удалось определить.
	PublicationStatus string
	Chapters          []Chapter
}

// ErrMangaRemoved — страница манги, удалённой или скрытой с сайта.
// Движок Grouple отдаёт такие страницы с HTTP 200 и пометкой в <title>
// («Манга ... противоречит правилам сайта»), подтверждено probe -url.
var ErrMangaRemoved = errors.New("манга удалена или скрыта с сайта")
