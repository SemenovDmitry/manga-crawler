// Package parsers — контракты и общие типы драйверов источников.
//
// Каждый сайт (или семейство сайтов на одном движке) реализует свой
// драйвер в подпакете (grouple, mgeko); краулер, расписание и уведомления
// от движка не зависят. Общие для всех драйверов сущности — карточка
// манги, глава, статусы публикации — собраны здесь, чтобы подпакеты
// зависели только от parsers, а друг от друга — нет.
package parsers

import "errors"

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
