// Package mangalib — драйвер сайта MangaLib (https://mangalib.me).
//
// Сайт — Vue SPA с выборочным SSR (подтверждено разведкой, см.
// probe-output/mangalib.me): браузерному User-Agent отдаётся пустой
// каркас ~8 КБ на всех страницах, а Googlebot-UA — полноценный SSR
// (~70 КБ) с JSON-LD и инфоблоком карточки. Поэтому драйвер реализует
// parsers.CardUserAgent: карточки загружаются с роботным UA, который
// сайт сам разрешает в robots.txt (отдельная секция Googlebot).
//
// Ключевые особенности:
//
//  1. /sitemap.xml — SPA-заглушка (HTML 200). Карты сайта перечислены
//     в robots.txt (Sitemap: /sitemap/manga-1..3); их подхватывает
//     фолбэк sitemap-листера краулера. В картах — прямые адреса
//     карточек /ru/manga/<id>--<slug> (~57,5 тыс. тайтлов, без глав).
//  2. Данные карточки — JSON-LD (CreativeWorkSeries): name,
//     alternativeHeadline, тип в identifier. Статус публикации и
//     счётчик глав — в текстовом инфоблоке («Статус»/«Онгоинг»,
//     «Глав»/число), классы-хеши нестабильны, поэтому ищем по меткам.
//  3. Списка глав в SSR нет. Главы догружаются из публичного API
//     (parsers.ChapterLoader): GET api.imglib.info/api/manga/{slug}/chapters
//     одним запросом (slug — часть URL после "--").
package mangalib

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers"
)

// googlebotUA — User-Agent, под которым сайт отдаёт полный SSR.
// Сайт сам разрешает роботов в robots.txt (секция Googlebot),
// подмена здесь — способ получить ту же страницу, что видит поисковик.
const googlebotUA = "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"

var (
	// JSON-LD CreativeWorkSeries: данные карточки (название, альтернативы).
	reLDJSON = regexp.MustCompile(`(?s)ld\+json">(.*?)</script>`)

	// Текстовые метки инфоблока. Классы-хеши (bel_*, bf5_*) меняются между
	// релизами, поэтому привязываемся к неизменным русским меткам; окно
	// в 200 байт не даёт перескочить в соседний блок.
	reStatusBlock = regexp.MustCompile(`(?s)Статус</div>.{0,200}?<span>([^<]+)</span>`)

	// Fallback-заголовок из SSR-разметки (h1 с единственным span).
	reH1Span = regexp.MustCompile(`(?s)<h1[^>]*>.*?<span>(.*?)</span>`)
)

// Parser — драйвер mangalib.me.
type Parser struct{}

// NewParser создаёт драйвер MangaLib.
func NewParser() *Parser { return &Parser{} }

// Compile-time проверки: контракт SourceDriver, роботный UA для карточек,
// догрузка глав из API.
var (
	_ parsers.SourceDriver   = (*Parser)(nil)
	_ parsers.CardUserAgent  = (*Parser)(nil)
	_ parsers.ChapterLoader  = (*Parser)(nil)
)

// CardUserAgent отдаёт роботный UA: сайт отдаёт полный SSR только роботам.
func (p *Parser) CardUserAgent() string { return googlebotUA }

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

// ParseMangaCard разбирает SSR-страницу карточки: JSON-LD (название,
// альтернативные названия) + текстовый инфоблок (статус). Списка глав
// в SSR нет — их догружает LoadChapters из API.
func (p *Parser) ParseMangaCard(body []byte) (*parsers.MangaCard, error) {
	page := string(body)

	series := extractLDJSON(page)

	title := ""
	alt := ""
	if series != nil {
		title = parsers.CleanTitle(series.Name)
		alt = joinAltTitles(series.AlternativeHeadline, title)
	}
	if title == "" {
		// Fallback: h1 SSR-разметки (JSON-LD мог сменить формат).
		if m := reH1Span.FindStringSubmatch(page); m != nil {
			title = parsers.CleanTitle(m[1])
		}
	}
	if title == "" {
		return nil, fmt.Errorf("страница не является карточкой манги mangalib (SPA-каркас или неизвестная разметка)")
	}

	return &parsers.MangaCard{
		Title:             title,
		AltTitles:         alt,
		PublicationStatus: extractStatus(page),
	}, nil
}

// ldSeries — поля CreativeWorkSeries из JSON-LD, нужные краулеру
// (у BreadcrumbList поля Name/Alt пустые — фильтруются по Type/Name).
type ldSeries struct {
	Type                string   `json:"@type"`
	Name                string   `json:"name"`
	AlternativeHeadline []string `json:"alternativeHeadline"`
}

// extractLDJSON достаёт объект CreativeWorkSeries из JSON-LD массива
// (первый элемент — BreadcrumbList, он игнорируется). nil — данных нет.
func extractLDJSON(page string) *ldSeries {
	for _, m := range reLDJSON.FindAllStringSubmatch(page, -1) {
		var items []ldSeries
		if err := json.Unmarshal([]byte(m[1]), &items); err != nil {
			continue // не JSON или не массив — пробуем следующий блок
		}
		for _, item := range items {
			if item.Name == "" || item.Type != "CreativeWorkSeries" {
				continue
			}
			series := item
			return &series
		}
	}
	return nil
}

// joinAltTitles собирает альтернативные названия из alternativeHeadline,
// отбрасывая дубли и совпадения с основным названием.
func joinAltTitles(alts []string, title string) string {
	var result []string
	seen := make(map[string]bool)
	for _, alt := range alts {
		alt = parsers.CleanTitle(alt)
		if alt == "" || alt == title || seen[alt] {
			continue
		}
		seen[alt] = true
		result = append(result, alt)
	}
	return strings.Join(result, " / ")
}

// extractStatus определяет статус публикации по текстовой метке
// «Статус» в инфоблоке карточки. Значения (catalog-фильтры mangalib):
// Онгоинг (1), Завершён (2), Заморожен (3), Анонс (4).
func extractStatus(page string) string {
	m := reStatusBlock.FindStringSubmatch(page)
	if m == nil {
		return ""
	}

	switch strings.ToLower(strings.TrimSpace(m[1])) {
	case "онгоинг":
		return "ongoing"
	case "завершён":
		return "completed"
	case "заморожен":
		return "hiatus"
	case "заброшен":
		return "discontinued"
	case "анонс":
		return ""
	default:
		return "" // неизвестное значение — статус не меняем
	}
}

// ——— догрузка глав из API ———

// ChapterLoader дополняет карточку главами из публичного API
// api.imglib.info: один запрос на тайтл, все главы сразу.
func (p *Parser) LoadChapters(ctx context.Context, load parsers.PageLoader, urlPart string, card *parsers.MangaCard) error {
	slug := slugFromURLPart(urlPart)
	if slug == "" {
		return fmt.Errorf("не удалось извлечь slug из url-части %q", urlPart)
	}

	body, err := load(ctx, "https://api.imglib.info/api/manga/"+url.PathEscape(slug)+"/chapters")
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

// slugFromURLPart извлекает slug тайтла из url-части
// ("ru/manga/123--slug" -> "slug").
func slugFromURLPart(urlPart string) string {
	parts := strings.Split(strings.Trim(urlPart, "/"), "/")
	if len(parts) != 3 || parts[0] != "ru" || parts[1] != "manga" {
		return ""
	}
	pair := strings.SplitN(parts[2], "--", 2)
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
