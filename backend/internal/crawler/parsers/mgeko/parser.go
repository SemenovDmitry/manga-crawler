package mgeko

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers"
)

var (
	reMgekoTitle   = regexp.MustCompile(`(?s)<h1[^>]+class="[^"]*novel-title[^"]*"[^>]*>(.*?)</h1>`)
	reMgekoAlt     = regexp.MustCompile(`(?s)<h2[^>]+class="[^"]*alternative-title[^"]*"[^>]*>(.*?)</h2>`)
	reMgekoStatus  = regexp.MustCompile(`(?is)<strong class="([a-z-]+)">[^<]*</strong>\s*<small>[^<]*Status`)
	reMgekoChapter = regexp.MustCompile(`(?s)<li[^>]*data-volumeno="([^"]*)"[^>]*>.*?<a href="/reader/[^"]*?-chapter-([0-9.]+)(-[0-9]+)?[^"]*"`)
	reMgekoAltPart = regexp.MustCompile(`[•/|]`)
)

// Parser — драйвер mgeko.cc.
type Parser struct{}

// Compile-time проверки: драйвер реализует контракт SourceDriver и умеет
// догружать полный список глав (карточка отдаёт только 50 последних).
var (
	_ parsers.SourceDriver  = (*Parser)(nil)
	_ parsers.ChapterLoader = (*Parser)(nil)
)

// NewParser создаёт драйвер mgeko.
func NewParser() *Parser { return &Parser{} }

// ParseMangaCard разбирает HTML страницы манги mgeko.
func (d *Parser) ParseMangaCard(body []byte) (*parsers.MangaCard, error) {
	page := string(body)

	title := extractMgekoTitle(page)
	if title == "" {
		return nil, fmt.Errorf("страница не является карточкой манги mgeko (заголовок не найден)")
	}

	return &parsers.MangaCard{
		Title:             title,
		AltTitles:         extractMgekoAltTitles(page),
		PublicationStatus: extractMgekoStatus(page),
		Chapters:          extractMgekoChapters(page),
	}, nil
}

// IsMangaURL — страница манги: ровно два сегмента пути "manga/<slug>",
// без расширения. Подстраницы вида /manga/<slug>/all-chapters/ отсекаются.
func (d *Parser) IsMangaURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(segments) != 2 || segments[0] != "manga" || segments[1] == "" {
		return false
	}

	return !strings.Contains(segments[1], ".")
}

// MangaURLPart — url-часть манги: путь целиком ("manga/j3pu-shadow-slave").
func (d *Parser) MangaURLPart(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("некорректный адрес %s: %w", rawURL, err)
	}

	return strings.Trim(u.Path, "/"), nil
}

// extractMgekoTitle возвращает основное название манги.
func extractMgekoTitle(page string) string {
	match := reMgekoTitle.FindStringSubmatch(page)
	if match == nil {
		return ""
	}

	return parsers.CleanTitle(match[1])
}

// extractMgekoAltTitles собирает альтернативные названия из блока
// alternative-title ("섀도우 슬레이브 • Shadow Slave"), дубли отбрасываются.
func extractMgekoAltTitles(page string) string {
	match := reMgekoAlt.FindString(page)
	if match == "" {
		return ""
	}

	var titles []string
	seen := make(map[string]bool)
	for _, part := range reMgekoAltPart.Split(parsers.CleanTitle(match), -1) {
		part = strings.TrimSpace(part)
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		titles = append(titles, part)
	}

	return strings.Join(titles, " / ")
}

// extractMgekoStatus определяет статус публикации по классу блока Status.
func extractMgekoStatus(page string) string {
	match := reMgekoStatus.FindStringSubmatch(page)
	if match == nil {
		return ""
	}

	return mapEngineStatus(match[1])
}

// mapEngineStatus переводит класс блока Status движка mgeko
// (ongoing, completed, hiatus, dropped, cancelled) в publication_status
// манги. Словарь принадлежит только этому движку: при добавлении нового
// источника со своими статусами маппится отдельный словарь в его драйвере.
func mapEngineStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ongoing":
		return "ongoing"
	case "completed":
		return "completed"
	case "hiatus":
		return "hiatus"
	case "dropped":
		return "discontinued"
	case "cancelled":
		return "cancelled"
	default:
		return "" // неизвестное значение — статус не меняем
	}
}

// extractMgekoChapters извлекает главы из списка на странице: номер —
// из href ("-chapter-<номер>-"), том — из data-volumeno ("0" — без тома).
func extractMgekoChapters(page string) []parsers.Chapter {
	matches := reMgekoChapter.FindAllStringSubmatch(page, -1)
	if len(matches) == 0 {
		return nil
	}

	chapters := make([]parsers.Chapter, 0, len(matches))
	for _, m := range matches {
		volume := strings.TrimSpace(m[1])
		if volume == "0" {
			volume = ""
		}
		chapters = append(chapters, parsers.Chapter{
			Volume: volume,
			Number: chapterURLNumberToFloat(m[2], m[3]),
		})
	}

	return chapters
}

// chapterURLNumberToFloat собирает номер главы из групп регекса: base —
// число перед необязательной частью (m[2] = "36"), part — сама часть
// («-1» в URL ...-chapter-36-1-eng-li, иначе ""). Части «36-1» становятся
// дробными номерами «36.1»: «36-1» как номер дублировал бы главу 36
// и ломал сортировку. Обычные номера («12», «12.5») не меняются.
func chapterURLNumberToFloat(base, part string) string {
	if part == "" {
		return base
	}

	return base + "." + strings.TrimPrefix(part, "-")
}

// mgekoBaseURL — базовый адрес источника: драйвер привязан к одному
// сайту (ср. remanga с жёстким адресом API). Слаг приходит из БД
// («manga/<slug>»), поэтому достаточно склейки путей.
const mgekoBaseURL = "https://www.mgeko.cc/"

// allChaptersURLPart — сегмент служебной страницы с полным списком глав.
const allChaptersURLPart = "all-chapters"

// LoadChapters догружает полный список глав со страницы /all-chapters/.
// SSR-карточка mgeko отдаёт только 50 последних глав, полный список живёт
// на отдельной странице с той же разметкой списка. Догрузка — одна
// страница; добавленная история (главы старее карточных 50) не должна
// считаться «новыми главами»: краулер отфильтровывает её по последнему
// известному номеру, см. parsers.FindNewChapters.
func (d *Parser) LoadChapters(ctx context.Context, load parsers.PageLoader, urlPart string, card *parsers.MangaCard) error {
	allChaptersURL := mgekoBaseURL + strings.Trim(urlPart, "/") + "/" + allChaptersURLPart + "/"

	body, err := load(ctx, allChaptersURL)
	if err != nil {
		return fmt.Errorf("страница all-chapters недоступна: %w", err)
	}

	chapters := extractMgekoChapters(string(body))
	if len(chapters) == 0 {
		return fmt.Errorf("на странице all-chapters не найдено глав")
	}

	// Карточные 50 глав входят и в полный список: дубли по номеру дают
	// завышенный ChaptersCount в БД. Оставляем первое вхождение номера
	// (в списке глав-части идут после основной главы — порядок верный).
	seen := make(map[string]bool, len(card.Chapters)+len(chapters))
	for _, ch := range card.Chapters {
		seen[ch.Volume+"|"+ch.Number] = true
	}
	for _, ch := range chapters {
		key := ch.Volume + "|" + ch.Number
		if seen[key] {
			continue
		}
		seen[key] = true
		card.Chapters = append(card.Chapters, ch)
	}

	return nil
}
