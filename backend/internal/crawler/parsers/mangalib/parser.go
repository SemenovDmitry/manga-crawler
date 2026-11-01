// Package mangalib — драйвер сайта MangaLib (https://mangalib.me).
//
// Сайт закрыт антибот-защитой DDoS-Guard: роботные User-Agent получают
// 503 (проверено 2026-11-01), а браузерным отдаётся пустой Vue-каркас
// без данных. Раньше карточки брали из SSR-страницы под Googlebot-UA —
// DDoS-Guard закрыл и этот путь. Поэтому драйвер реализует
// parsers.CardRequestProvider: данные карточки и главы берутся из API
// api.imglib.info (тем же, каким пользуется фронтенд сайта), который
// защитой не прикрыт и пускает с браузерными заголовками mangalib.me:
//
//   - карточка: GET api.imglib.info/api/manga/{id}--{slug}
//   - главы:    GET api.imglib.info/api/manga/{slug}/chapters
//
// Заголовки (проверено 2026-11-01): Referer https://mangalib.me/ обязателен
// (без него 403), Site-Id: 1 выбирает базу mangalib (без него 404 — API
// мультисайтовый, обслуживает и ranobelib и др.).
//
// Прочее:
//
//  1. /sitemap.xml — SPA-заглушка (HTML 200). Карты сайта перечислены
//     в robots.txt (Sitemap: /sitemap/manga-1..3); их подхватывает
//     фолбэк sitemap-листера краулера. В картах — прямые адреса
//     карточек /ru/manga/<id>--<slug> (~57,5 тыс. тайтлов).
//  2. Главы есть не у всех типов контента: у «Руманга» и додзинси
//     эндпоинт /chapters стабильно отвечает 404 (2026-11-01) — карточка
//     сохраняется без глав, это не сбой.
package mangalib

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers"
)

// Данные запросов к API: те же, что шлёт браузер на mangalib.me.
// siteID — идентификатор сайта mangalib в мультисайтовом API LibGroup.
const (
	apiBase    = "https://api.imglib.info/api"
	apiReferer = "https://mangalib.me/"
	siteID     = "1"
)

var (
	// Заголовки для любого запроса к API: Referer обязателен (403),
	// Site-Id выбирает сайт (404 без него).
	apiHeaders = []parsers.ExtraHeader{
		{Key: "Referer", Value: apiReferer},
		{Key: "Site-Id", Value: siteID},
	}
)

// Parser — драйвер mangalib.me.
type Parser struct{}

// NewParser создаёт драйвер MangaLib.
func NewParser() *Parser { return &Parser{} }

// Compile-time проверки: контракт SourceDriver, карточка и главы из API.
var (
	_ parsers.SourceDriver        = (*Parser)(nil)
	_ parsers.CardRequestProvider = (*Parser)(nil)
	_ parsers.ChapterLoader       = (*Parser)(nil)
)

// CardRequest отдаёт адрес карточки в API (id--slug — третий сегмент
// url-части "ru/manga/274927--kot-baiun") и обязательные заголовки.
func (p *Parser) CardRequest(urlPart string) (string, []parsers.ExtraHeader) {
	idSlug := idSlugFromURLPart(urlPart)
	if idSlug == "" {
		return "", nil
	}
	return apiBase + "/manga/" + url.PathEscape(idSlug), apiHeaders
}

// IsMangaURL — страница манги: /ru/manga/<id>--<slug> (ровно три сегмента
// пути, второй — "ru", третий начинается с "manga/"). Главы читалки
// (/ru/<id>--<slug>/read/...) и каталог в sitemap не попадают.
func (p *Parser) IsMangaURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(segments) != 3 || segments[0] != "ru" || segments[1] != "manga" {
		return false
	}

	// <id>--<slug>: числовой префикс и разделитель обязательны.
	parts := strings.SplitN(segments[2], "--", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	for _, r := range parts[0] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// MangaURLPart — url-часть манги: путь целиком ("ru/manga/123--slug").
func (p *Parser) MangaURLPart(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("некорректный адрес %s: %w", rawURL, err)
	}

	return strings.Trim(u.Path, "/"), nil
}

// ParseMangaCard разбирает JSON-ответ API карточки (см. CardRequest):
// название и альтернативное английское, статус публикации из label.
// Списка глав в карточке нет — их догружает LoadChapters.
func (p *Parser) ParseMangaCard(body []byte) (*parsers.MangaCard, error) {
	var card apiCard
	if err := json.Unmarshal(body, &card); err != nil {
		return nil, fmt.Errorf("карточка mangalib: ответ не JSON: %w", err)
	}

	title := parsers.CleanTitle(card.Data.Name)
	if title == "" {
		title = parsers.CleanTitle(card.Data.RusName)
	}
	if title == "" {
		return nil, fmt.Errorf("карточка mangalib без названия (неизвестный формат ответа API)")
	}

	alt := parsers.CleanTitle(card.Data.EngName)
	if alt == title {
		alt = ""
	}

	return &parsers.MangaCard{
		Title:             title,
		AltTitles:         alt,
		PublicationStatus: statusFromLabel(card.Data.Status.Label),
	}, nil
}

// apiCard — ответ GET /api/manga/{id}--{slug}: интересуют название,
// английское название и статус публикации.
type apiCard struct {
	Data struct {
		Name    string `json:"name"`     // основное название (рус)
		RusName string `json:"rus_name"` // дублирует name
		EngName string `json:"eng_name"` // альтернативное (англ)
		Status  struct {
			Label string `json:"label"` // «Онгоинг», «Завершён», ...
		} `json:"status"`
	} `json:"data"`
}

// statusFromLabel переводит label статуса API в PublicationStatus.
// Значения совпадают с фильтрами каталога mangalib:
// Онгоинг (1), Завершён (2), Заморожен (3), Анонс (4).
func statusFromLabel(label string) string {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "онгоинг":
		return "ongoing"
	case "завершён":
		return "completed"
	case "заморожен":
		return "hiatus"
	case "заброшен":
		return "discontinued"
	default:
		return "" // «Анонс» и неизвестное — статус не меняем
	}
}

// ——— догрузка глав из API ———

// ChapterLoader дополняет карточку главами из API: один запрос на тайтл,
// все главы сразу. У «Руманга» и додзинси глав в API нет (404) — карточка
// сохранится без глав, это не сбой.
func (p *Parser) LoadChapters(ctx context.Context, load parsers.PageLoader, urlPart string, card *parsers.MangaCard) error {
	slug := slugFromURLPart(urlPart)
	if slug == "" {
		return fmt.Errorf("не удалось извлечь slug из url-части %q", urlPart)
	}

	body, err := load(ctx, apiBase+"/manga/"+url.PathEscape(slug)+"/chapters", apiHeaders...)
	if err != nil {
		return fmt.Errorf("главы %s из API недоступны: %w", slug, err)
	}

	chapters, err := parseChaptersAPI(body)
	if err != nil {
		return fmt.Errorf("главы %s не разобраны: %w", slug, err)
	}

	card.Chapters = append(card.Chapters, chapters...)
	return nil
}

// idSlugFromURLPart извлекает сегмент "id--slug" из url-части
// ("ru/manga/274927--kot-baiun" -> "274927--kot-baiun").
func idSlugFromURLPart(urlPart string) string {
	parts := strings.Split(strings.Trim(urlPart, "/"), "/")
	if len(parts) != 3 || parts[0] != "ru" || parts[1] != "manga" {
		return ""
	}
	return parts[2]
}

// slugFromURLPart извлекает slug тайтла из url-части
// ("ru/manga/123--slug" -> "slug").
func slugFromURLPart(urlPart string) string {
	pair := strings.SplitN(idSlugFromURLPart(urlPart), "--", 2)
	if len(pair) != 2 || pair[1] == "" {
		return ""
	}
	return pair[1]
}

// apiChapters — ответ GET /api/manga/{slug}/chapters: {"data": [...]}.
type apiChapters struct {
	Data []struct {
		Volume string `json:"volume"`
		Number string `json:"number"`
		Name   string `json:"name"`
	} `json:"data"`
}

// parseChaptersAPI разбирает ответ API в список глав.
func parseChaptersAPI(body []byte) ([]parsers.Chapter, error) {
	var resp apiChapters
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	chapters := make([]parsers.Chapter, 0, len(resp.Data))
	for _, ch := range resp.Data {
		chapters = append(chapters, parsers.Chapter{
			Number: strings.TrimSpace(ch.Number),
			Volume: strings.TrimSpace(ch.Volume),
			Title:  strings.TrimSpace(ch.Name),
		})
	}
	return chapters, nil
}
