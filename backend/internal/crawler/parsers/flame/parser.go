// Package flame — драйвер сайта Flame Comics (https://flamecomics.xyz).
//
// Сайт — самописный Next.js (pages router) с Mantine UI. Разметка карточки
// использует классы Mantine с хешами версии (m_8a5d1357, SeriesPage-module__zgaxnW),
// поэтому парсинг строится на данных __NEXT_DATA__ (структурированный JSON
// с title, altTitles, status и полным списком глав) и стабильных признаках
// (mantine-Title-root, href="/series/<id>/<hex>"), а не на хешах классов.
//
// Все страницы манги лежат по адресам /series/<id> с числовым id без слага;
// в sitemap (https://flamecomics.xyz/sitemaps/series.xml) ~154 URL — и это
// полный каталог: /browse отдаёт те же серии одним списком без пагинации,
// поэтому драйвер не реализует CatalogLister (перечисление только по sitemap).
package flame

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers"
)

var (
	// Fallback-разметка: заголовок карточки — единственный h1 mantine-Title-root.
	reTitle = regexp.MustCompile(`(?s)<h1[^>]*mantine-Title-root[^>]*>(.*?)</h1>`)
	// Fallback-главы: ссылки вида href="/series/2/364db6fd6bef182e" — серия + hex-токен.
	reChapter = regexp.MustCompile(`href="/series/\d+/([0-9a-f]{16})"`)
	// Страница-заглушка для отсутствующей серии отдаёт 404 (проверено probe),
	// поэтому отдельного маркера «удалено» в 200-ответе не ищем.
)

// nextData — верхний уровень __NEXT_DATA__ (важны только props.pageProps).
type nextData struct {
	Props struct {
		PageProps struct {
			Series    rawSeries  `json:"series"`
			Chapters  []rawChapter `json:"chapters"`
		} `json:"pageProps"`
	} `json:"props"`
}

// rawSeries — карточка серии из pageProps.series (проверено probe на /series/2).
type rawSeries struct {
	Title     string   `json:"title"`
	AltTitles []string `json:"altTitles"`
	Status    string   `json:"status"`
}

// rawChapter — глава из pageProps.chapters; томов сайт не знает, только номер.
type rawChapter struct {
	Chapter string `json:"chapter"` // "311.00" — float с двумя знаками
	Title   string `json:"title"`
}

// Parser — драйвер flamecomics.xyz.
type Parser struct{}

// NewParser создаёт драйвер Flame Comics.
func NewParser() *Parser { return &Parser{} }

// Compile-time проверка: драйвер реализует контракт SourceDriver.
var _ parsers.SourceDriver = (*Parser)(nil)

// IsMangaURL — страница манги: /series/<id>, где id — число без нулей впереди
// (категории, новеллы, блог и статика отсекаются; отдельные /genre/... и
// /novel/<id> не проходят по префиксу).
func (p *Parser) IsMangaURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(segments) != 2 || segments[0] != "series" {
		return false
	}

	id := segments[1]
	if id == "" {
		return false
	}
	n, err := strconv.Atoi(id)
	return err == nil && n > 0 && strconv.Itoa(n) == id
}

// MangaURLPart — url-часть манги: путь целиком ("series/2").
func (p *Parser) MangaURLPart(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("некорректный адрес %s: %w", rawURL, err)
	}

	return strings.Trim(u.Path, "/"), nil
}

// ParseMangaCard разбирает HTML страницы серии. Основной путь — JSON из
// __NEXT_DATA__ (структура подтверждена probe: pageProps.series.title,
// .status, pageProps.chapters[].chapter); при его отсутствии — fallback
// на видимую разметку (h1 mantine-Title-root + ссылки глав).
func (p *Parser) ParseMangaCard(body []byte) (*parsers.MangaCard, error) {
	if card, ok := parseNextData(body); ok {
		return card, nil
	}

	// Fallback: страница без __NEXT_DATA__ (изменение SSR или уже другая
	// структура). Если и тут заголовка нет — не карточка, уйдёт в bad-page дамп.
	title := extractTitleFromMarkup(body)
	if title == "" {
		return nil, fmt.Errorf("страница не является карточкой манги flame (заголовок не найден)")
	}

	return &parsers.MangaCard{
		Title:    title,
		Chapters: extractChaptersFromMarkup(body),
	}, nil
}

// parseNextData извлекает карточку из __NEXT_DATA__; ok=false — скрипта нет
// или структура не распознана (пустой title).
func parseNextData(body []byte) (*parsers.MangaCard, bool) {
	// Скрипт с JSON без вложенных <script>, поэтому жадный поиск до </script>
	// безопасен; сам JSON содержит экранированные </script> как \u003c/script\u003e.
	start := strings.Index(string(body), `id="__NEXT_DATA__"`)
	if start < 0 {
		return nil, false
	}
	rest := body[start:]

	open := strings.Index(string(rest), ">")
	if open < 0 {
		return nil, false
	}
	rest = rest[open+1:]

	end := strings.Index(string(rest), "</script>")
	if end < 0 {
		return nil, false
	}

	var data nextData
	if err := json.Unmarshal(rest[:end], &data); err != nil {
		return nil, false
	}

	series := data.Props.PageProps.Series
	if strings.TrimSpace(series.Title) == "" {
		return nil, false
	}

	return &parsers.MangaCard{
		Title:             parsers.CleanTitle(series.Title),
		AltTitles:         joinAltTitles(series.AltTitles),
		PublicationStatus: mapStatus(series.Status),
		Chapters:          chaptersFromJSON(data.Props.PageProps.Chapters),
	}, true
}

// joinAltTitles приводит список альтернативных названий к строке через " / "
// (формат MangaCard.AltTitles других драйверов), дубли отбрасываются.
func joinAltTitles(alts []string) string {
	var titles []string
	seen := make(map[string]bool)
	for _, alt := range alts {
		alt = parsers.CleanTitle(alt)
		if alt == "" || seen[alt] {
			continue
		}
		seen[alt] = true
		titles = append(titles, alt)
	}
	return strings.Join(titles, " / ")
}

// chaptersFromJSON переводит главы из JSON в parsers.Chapter: номер
// "311.00" нормализуется до "311" (иначе NormalizeChapters и сравнение
// latest-chapter в БД увидят "311.00" и "311" разными главами).
func chaptersFromJSON(raw []rawChapter) []parsers.Chapter {
	if len(raw) == 0 {
		return nil
	}

	chapters := make([]parsers.Chapter, 0, len(raw))
	for _, rc := range raw {
		chapters = append(chapters, parsers.Chapter{
			Number: normalizeChapterNumber(rc.Chapter),
			Title:  strings.TrimSpace(rc.Title),
		})
	}

	return chapters
}

// normalizeChapterNumber убирает дробную часть "311.00" -> "311", сохраняя
// значимые десятые "12.50" -> "12.5"; пустое или нечисловое значение
// возвращается как есть (normalizeChapters отсортирует по остальным).
func normalizeChapterNumber(raw string) string {
	raw = strings.TrimSpace(raw)
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return raw
	}

	number := strconv.FormatFloat(value, 'f', -1, 64)
	return number
}

// extractTitleFromMarkup — fallback-заголовок: первый h1 mantine-Title-root.
func extractTitleFromMarkup(body []byte) string {
	match := reTitle.FindStringSubmatch(string(body))
	if match == nil {
		return ""
	}
	return parsers.CleanTitle(match[1])
}

// extractChaptersFromMarkup — fallback-главы: ссылки на главы по маске
// href="/series/<id>/<16 hex-символов>" (включая кнопки First/Latest —
// дубли убираются по токену). Номера глав в разметке разорваны
// HTML-комментариями React ("Chapter <!-- -->311<!-- -->"), поэтому
// fallback-главы без номера — нужен только счётчик глав (FindNewChapters
// сравнивает количество).
func extractChaptersFromMarkup(body []byte) []parsers.Chapter {
	matches := reChapter.FindAllStringSubmatch(string(body), -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	chapters := make([]parsers.Chapter, 0, len(matches))
	for _, m := range matches {
		token := m[1]
		if seen[token] {
			continue
		}
		seen[token] = true
		chapters = append(chapters, parsers.Chapter{})
	}

	return chapters
}

// mapStatus переводит статус движка Flame ("Ongoing", "Completed", "Hiatus",
// "Dropped", "Cancelled", "Coming Soon" — все подтверждены в выдаче /browse)
// в publication_status манги. Словарь принадлежит только этому движку.
func mapStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ongoing":
		return "ongoing"
	case "completed":
		return "completed"
	case "hiatus":
		return "hiatus"
	case "cancelled":
		return "cancelled"
	case "dropped":
		return "discontinued"
	default:
		return "" // "coming soon" и неизвестные значения — статус не меняем
	}
}
