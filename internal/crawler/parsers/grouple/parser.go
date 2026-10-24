package grouple

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers"
)

var (
	reMangaTitle  = regexp.MustCompile(`(?s)<h1 class="cr-hero-names__main[^"]*">\s*(.*?)\s*</h1>`)
	reAltBlock    = regexp.MustCompile(`(?s)<h3 class="cr-hero-names__alt.*?</h3>`)
	reAltSpan     = regexp.MustCompile(`<span>(.*?)</span>`)
	reChapterLink = regexp.MustCompile(`(?s)<a href="/[^"]*?/vol(\d+)/([0-9.-]+)"[^>]*?class="chapter-link cp-l[^"]*"[^>]*>(.*?)</a>`)
	reChapterPref = regexp.MustCompile(`^\s*\d+\s*-\s*[0-9.]+\s*`)
	reRemovedPage = regexp.MustCompile(`(?i)<title>[^<]*противоречит правилам[^<]*</title>`)

	// Статусы в блоке деталей карточки.
	reTranslationStatus = regexp.MustCompile(`(?i)data-translation-status="([a-z]+)"`)
	reProductionStatus  = regexp.MustCompile(`(?i)data-production-status="([a-z]+)"`)

	// Каталог /list: карточки — ссылки class="non-hover" внутри тайлов;
	// на странице ровно 50 карточек (подтверждено probe для seimanga и zazaza).
	reCatalogMangaLink = regexp.MustCompile(`<a href="/([^"/?#]+)" class="non-hover">`)
)

// groupleCatalogPageSize — число карточек на странице каталога /list
// (подтверждено probe: ровно 50 у seimanga и zazaza).
const groupleCatalogPageSize = 50

// Parser — драйвер сайтов на движке Grouple.
type Parser struct{}

// NewParser создаёт драйвер Grouple.
func NewParser() *Parser { return &Parser{} }

// Compile-time проверка: драйвер умеет обходить каталог.
var _ parsers.CatalogLister = (*Parser)(nil)

// IsMangaURL — правила URL движка Grouple: страница манги — один сегмент
// пути без расширения и служебных префиксов.
func (p *Parser) IsMangaURL(rawURL string) bool {
	return isMangaSitemapURL(rawURL)
}

// MangaURLPart — правила URL движка Grouple: url-часть — путь целиком
// (у карточек он состоит из одного сегмента).
func (p *Parser) MangaURLPart(rawURL string) (string, error) {
	return mangaURLPart(rawURL)
}

// ParseMangaCard разбирает HTML страницы манги: название, альтернативные
// названия и список глав. Для удалённых/скрытых манг возвращает
// parsers.ErrMangaRemoved, для остальных не-карточек — обычную ошибку.
func (p *Parser) ParseMangaCard(body []byte) (*parsers.MangaCard, error) {
	page := string(body)

	if reRemovedPage.MatchString(page) {
		return nil, parsers.ErrMangaRemoved
	}

	title := extractMangaTitle(page)
	if title == "" {
		return nil, fmt.Errorf("страница не является карточкой манги (заголовок не найден)")
	}

	return &parsers.MangaCard{
		Title:             title,
		AltTitles:         extractAltTitles(page),
		PublicationStatus: extractPublicationStatus(page),
		Chapters:          extractChapters(page),
	}, nil
}

// CatalogPageURL — адрес страницы каталога, отсортированного по дате
// создания (новые манги сверху — этот же порядок используется
// инкрементальным проходом краулера).
func (p *Parser) CatalogPageURL(baseURL string, offset int) string {
	return fmt.Sprintf("%s/list?sortType=created&offset=%d", strings.TrimRight(baseURL, "/"), offset)
}

// CatalogPageSize — число карточек манги на одной странице каталога.
func (p *Parser) CatalogPageSize() int {
	return groupleCatalogPageSize
}

// MangaURLsFromCatalog извлекает URL карточек манги из HTML страницы
// каталога (ссылки class="non-hover" внутри тайлов) и возвращает их
// в исходном порядке страницы, без повторов. Служебные страницы
// отфильтровываются теми же правилами, что и URL из sitemap.
func (p *Parser) MangaURLsFromCatalog(body []byte, baseURL string) []string {
	base := strings.TrimRight(baseURL, "/")

	var urls []string
	seen := make(map[string]bool)
	for _, m := range reCatalogMangaLink.FindAllStringSubmatch(string(body), -1) {
		slug := m[1]
		if seen[slug] || !isMangaSitemapURL(base+"/"+slug) {
			continue
		}
		seen[slug] = true
		urls = append(urls, base+"/"+slug)
	}

	return urls
}

// extractPublicationStatus определяет статус публикации по атрибутам
// data-translation-status / data-production-status (подтверждено probe:
// "FINISHED" у завершённого «Ганца», "PROGRESS" у идущего «Ван Писа»).
func extractPublicationStatus(page string) string {
	if match := reTranslationStatus.FindStringSubmatch(page); match != nil {
		if status := mapEngineStatus(match[1]); status != "" {
			return status
		}
	}

	if match := reProductionStatus.FindStringSubmatch(page); match != nil {
		if status := mapEngineStatus(match[1]); status != "" {
			return status
		}
	}

	return ""
}

// mapEngineStatus переводит статус движка Grouple (атрибуты
// data-translation-status / data-production-status) в publication_status
// манги. Словарь принадлежит только этому движку: при добавлении нового
// источника со своими статусами маппится отдельный словарь в его драйвере.
// Перевод приоритетнее выхода оригинала: для читателя важно, выходят ли
// ещё переведённые главы.
func mapEngineStatus(value string) string {
	v := strings.ToUpper(strings.TrimSpace(value))

	switch {
	case strings.Contains(v, "FINISH"), strings.Contains(v, "COMPLETE"):
		return "completed" // «Перевод Завершён»
	case strings.Contains(v, "PAUSE"), strings.Contains(v, "FREEZE"), strings.Contains(v, "HIATU"):
		return "hiatus" // «Перевод Приостановлен»
	case strings.Contains(v, "CANCEL"):
		return "cancelled"
	case strings.Contains(v, "DISCONTIN"), strings.Contains(v, "ABANDON"), strings.Contains(v, "DROP"):
		return "discontinued"
	case strings.Contains(v, "PROGRESS"), strings.Contains(v, "ONGO"):
		return "ongoing"
	default:
		return "" // неизвестное значение — статус не меняем
	}
}

// extractMangaTitle возвращает основное название манги.
func extractMangaTitle(page string) string {
	match := reMangaTitle.FindStringSubmatch(page)
	if match == nil {
		return ""
	}

	return parsers.CleanTitle(match[1])
}

// extractAltTitles собирает альтернативные названия (обычные спаны в блоке
// cr-hero-names__alt; спаны-разделители с классами игнорируются).
func extractAltTitles(page string) string {
	match := reAltBlock.FindString(page)
	if match == "" {
		return ""
	}

	var titles []string
	seen := make(map[string]bool)
	for _, span := range reAltSpan.FindAllStringSubmatch(match, -1) {
		for _, part := range strings.Split(parsers.CleanTitle(span[1]), "/") {
			part = strings.TrimSpace(part)
			if part == "" || seen[part] {
				continue
			}
			seen[part] = true
			titles = append(titles, part)
		}
	}

	return strings.Join(titles, " / ")
}

// extractChapters извлекает главы из ссылок class="chapter-link cp-l".
func extractChapters(page string) []parsers.Chapter {
	matches := reChapterLink.FindAllStringSubmatch(page, -1)
	if len(matches) == 0 {
		return nil
	}

	chapters := make([]parsers.Chapter, 0, len(matches))
	for _, m := range matches {
		chapters = append(chapters, parsers.Chapter{
			Volume: m[1],
			Number: m[2],
			Title:  cleanChapterTitle(m[3]),
		})
	}

	return chapters
}

// cleanChapterTitle убирает теги и префикс "том - номер" из подписи главы.
func cleanChapterTitle(raw string) string {
	text := parsers.CleanTitle(raw)
	return strings.TrimSpace(reChapterPref.ReplaceAllString(text, ""))
}

// mangaURLPart извлекает url-часть манги из полного адреса ("asura_2024").
func mangaURLPart(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("некорректный адрес %s: %w", rawURL, err)
	}

	return strings.Trim(u.Path, "/"), nil
}

// isMangaSitemapURL проверяет, что URL из sitemap — это страница манги
// (один сегмент пути, без расширения и служебных префиксов).
func isMangaSitemapURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}

	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(segments) != 1 || segments[0] == "" {
		return false
	}

	segment := segments[0]
	if strings.Contains(segment, ".") {
		return false
	}

	excluded := map[string]bool{
		"list": true, "search": true, "rss": true, "api": true,
		"login": true, "random": true, "types": true, "genres": true,
		"about": true, "quote": true, "news": true, "review": true,
		"users": true, "teams": true, "blogs": true, "forum": true,
	}

	return !excluded[segment]
}
