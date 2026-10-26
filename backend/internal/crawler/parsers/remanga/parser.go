// Package flame — драйвер сайта Remanga (https://remanga.org).
//
// Сайт — собственный Next.js App Router + Tailwind, API на отдельном хосте
// api.remanga.org (Django REST, пути /api/...). Особенности, подтверждённые
// разведкой (probe, см. probe-output/remanga.org):
//
//  1. __NEXT_DATA__ нет: данные карточки лежат в RSC-payload
//     (self.__next_f.push) в виде экранированного JSON react-query кэша.
//     Полный объект тайтла — единственный в payload, у которого есть
//     "translate_status" (у карточек рекомендаций его нет) — по этому
//     якорю объект и находится.
//  2. Sitemap (~93 под-карты по ~50 тыс. URL) не содержит карточек:
//     только 7 подстраниц на тайтл (/main, /similar, /chapters, /posts,
//     /moments, /cards, /characters) и отдельные карты глав
//     (/manga/<slug>/<chapter_id>). URL карточки восстанавливается из
//     подстраницы: /manga/<slug>/main -> "manga/<slug>", а дубль
//     склеивается дедупликацией краулера по url-части.
//  3. Список глав в SSR отсутствует (только счётчик count_chapters и
//     первая глава). Главы догружаются из API (см. ChapterLoader):
//     GET api.remanga.org/api/titles/{dir}/          — ветки тайтла;
//     GET api.remanga.org/api/titles/chapters/?branch_id={id}&page={n}
//     — главы ветки, 30 штук на страницу, от новых к старым,
//     пустой "content" — конец списка.
package remanga

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers"
)

// Подстраницы тайтла, встречающиеся в sitemap. Прямых карточек
// /manga/<slug> в карте нет, но канонический адрес отвечает 200.
var titleSubpages = map[string]bool{
	"main": true, "similar": true, "chapters": true,
	"posts": true, "moments": true, "cards": true, "characters": true,
}

// Служебные "карточки" первого уровня: /manga/top, /manga/hot-updates.
var reservedSlugs = map[string]bool{
	"top": true, "hot-updates": true,
}

var (
	// Якорь основного объекта тайтла в RSC-payload: "translate_status"
	// есть только у карточки текущего тайтла, за ним идёт его dir.
	// Кавычки в регексах чистые: страницы с экранированным payload
	// (\") распаковываются перед разбором, см. extractTitleData.
	reTitleDir = regexp.MustCompile(`"translate_status":\{"id":\d+,"name":"[^"]*"\},"dir":"([a-z0-9_.-]+)"`)
	// Поля объекта тайтла. Значения — с учётом возможных \" и \uXXXX
	// внутри строки JSON.
	reMainName = regexp.MustCompile(`"main_name":"((?:[^"\\]|\\.)*)"`)
	reSecName  = regexp.MustCompile(`"secondary_name":"((?:[^"\\]|\\.)*)"`)
	reAltName  = regexp.MustCompile(`"another_name":"((?:[^"\\]|\\.)*)"`)
	// Статус публикации — блок "status" непосредственно перед
	// translate_status (см. порядок полей в payload).
	reStatus = regexp.MustCompile(`"status":\{"id":(\d+),"name":"((?:[^"\\]|\\.)*)"\},"translate_status"`)
	// Fallback-заголовок: единственный h1 на карточке (Tailwind-класс
	// семантический, в отличие от хешей utility-классов).
	reH1 = regexp.MustCompile(`(?s)<h1[^>]*cs-layout-title-text[^>]*>(.*?)</h1>`)
)

// Parser — драйвер remanga.org.
type Parser struct{}

// NewParser создаёт драйвер Remanga.
func NewParser() *Parser { return &Parser{} }

// Compile-time проверки: драйвер реализует контракт SourceDriver
// и умеет догружать главы из API.
var (
	_ parsers.SourceDriver   = (*Parser)(nil)
	_ parsers.ChapterLoader  = (*Parser)(nil)
)

// IsMangaURL — страница манги: /manga/<slug> или подстраница тайтла
// /manga/<slug>/<main|similar|...>. Главы (/manga/<slug>/<число>),
// служебные (/manga/top, /manga/hot-updates) и прочее отсекаются.
func (p *Parser) IsMangaURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(segments) < 2 || segments[0] != "manga" || segments[1] == "" {
		return false
	}

	slug := segments[1]
	if reservedSlugs[slug] {
		return false
	}

	switch len(segments) {
	case 2:
		return true // прямая карточка /manga/<slug>
	case 3:
		return titleSubpages[segments[2]] // подстраница тайтла из sitemap
	default:
		return false // главы и прочая вложенность
	}
}

// MangaURLPart — url-часть манги: путь целиком ("manga/slug").
func (p *Parser) MangaURLPart(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("некорректный адрес %s: %w", rawURL, err)
	}

	return strings.Trim(u.Path, "/"), nil
}

// ParseMangaCard разбирает HTML страницы тайтла: данные из RSC-payload,
// fallback заголовка — h1 разметки. Списка глав в SSR нет, главы
// догружает LoadChapters.
func (p *Parser) ParseMangaCard(body []byte) (*parsers.MangaCard, error) {
	page := string(body)

	title, alt, status := extractTitleData(page)
	if title == "" {
		// Payload не распознался: пробуем заголовок из разметки,
		// иначе страница уйдёт в bad-page дамп.
		if m := reH1.FindStringSubmatch(page); m != nil {
			title = parsers.CleanTitle(m[1])
		}
	}
	if title == "" {
		return nil, fmt.Errorf("страница не является карточкой манги remanga (тайтл в payload и h1 не найдены)")
	}

	return &parsers.MangaCard{
		Title:             title,
		AltTitles:         alt,
		PublicationStatus: status,
	}, nil
}

// LoadChapters догружает главы тайтла из API remanga: карточка по dir
// (первая ветка — основная команда перевода), затем главы постранично
// по 30 штук, от новых к старым, до пустой страницы.
func (p *Parser) LoadChapters(ctx context.Context, load parsers.PageLoader, urlPart string, card *parsers.MangaCard) error {
	dir := dirFromURLPart(urlPart)
	if dir == "" {
		return fmt.Errorf("не удалось извлечь dir из url-части %q", urlPart)
	}

	branchID, err := p.firstBranchID(ctx, load, dir)
	if err != nil {
		return err
	}

	chapters, err := p.fetchBranchChapters(ctx, load, branchID)
	if err != nil {
		return err
	}

	card.Chapters = append(card.Chapters, chapters...)
	return nil
}

// dirFromURLPart извлекает слаг тайтла из url-части ("manga/slug").
func dirFromURLPart(urlPart string) string {
	parts := strings.Split(strings.Trim(urlPart, "/"), "/")
	if len(parts) != 2 || parts[0] != "manga" {
		return ""
	}
	return parts[1]
}

// apiTitleCard — ответ GET /api/titles/{dir}/: интересны только ветки.
type apiTitleCard struct {
	Content struct {
		Branches []struct {
			ID            int `json:"id"`
			CountChapters int `json:"count_chapters"`
		} `json:"branches"`
	} `json:"content"`
}

// apiChaptersPage — ответ GET /api/titles/chapters/?branch_id=...&page=N.
type apiChaptersPage struct {
	Content []struct {
		ID      int    `json:"id"`
		Index   int    `json:"index"` // сквозной номер главы в ветке
		Tome    int    `json:"tome"`
		Chapter string `json:"chapter"`
		Name    string `json:"name"`
	} `json:"content"`
	Next *string `json:"next"`
}

// firstBranchID возвращает ID первой ветки тайтла из API.
func (p *Parser) firstBranchID(ctx context.Context, load parsers.PageLoader, dir string) (int, error) {
	body, err := load(ctx, "https://api.remanga.org/api/titles/"+url.PathEscape(dir)+"/")
	if err != nil {
		return 0, fmt.Errorf("карточка тайтла %s из API недоступна: %w", dir, err)
	}

	var card apiTitleCard
	if err := json.Unmarshal(body, &card); err != nil {
		return 0, fmt.Errorf("ответ API %s не разобран: %w", dir, err)
	}

	if len(card.Content.Branches) == 0 {
		return 0, fmt.Errorf("у тайтла %s нет веток перевода", dir)
	}

	return card.Content.Branches[0].ID, nil
}

// maxChaptersPages — защитный предел страниц API на один тайтл
// (30 глав на страницу; реальный максимум — сотни глав).
const maxChaptersPages = 40

// fetchBranchChapters собирает все главы ветки, идя по страницам API
// от новых глав к старым до пустого content.
func (p *Parser) fetchBranchChapters(ctx context.Context, load parsers.PageLoader, branchID int) ([]parsers.Chapter, error) {
	var chapters []parsers.Chapter

	for page := 1; ; page++ {
		if ctx.Err() != nil {
			return chapters, ctx.Err()
		}
		if page > maxChaptersPages {
			return chapters, fmt.Errorf("ветка %d: достигнут предел страниц API (%d)", branchID, maxChaptersPages)
		}

		apiURL := fmt.Sprintf("https://api.remanga.org/api/titles/chapters/?branch_id=%d&page=%d", branchID, page)
		body, err := load(ctx, apiURL)
		if err != nil {
			return chapters, fmt.Errorf("страница глав %d ветки %d: %w", page, branchID, err)
		}

		var data apiChaptersPage
		if err := json.Unmarshal(body, &data); err != nil {
			return chapters, fmt.Errorf("страница глав %d ветки %d не разобрана: %w", page, branchID, err)
		}

		if len(data.Content) == 0 {
			return chapters, nil // конец списка
		}

		for _, ch := range data.Content {
			chapters = append(chapters, parsers.Chapter{
				Number: strconv.Itoa(ch.Index), // сквозной номер главы ветки
				Volume: volumeLabel(ch.Tome),
				Title:  strings.TrimSpace(ch.Name),
			})
		}
	}
}

// volumeLabel превращает том главы в строку ("0" — без тома).
func volumeLabel(tome int) string {
	if tome <= 0 {
		return ""
	}
	return strconv.Itoa(tome)
}

// extractTitleData находит основной объект тайтла в RSC-payload: по якорю
// translate_status + dir вырезается окно, из которого достаются название,
// альтернативные названия и статус публикации.
//
// Payload встречается в двух видах (проверено на дампах): с чистыми
// кавычками (SSR-чанк в начале страницы) и с экранированными \" в
// hydration-скрипте в конце. Если чистый вариант не сматчился — пробуем
// страницу с распакованными кавычками.
func extractTitleData(page string) (title, alt, status string) {
	title, alt, status = extractTitleDataVariant(page)
	if title != "" {
		return title, alt, status
	}
	if !strings.Contains(page, `\"`) {
		return "", "", "" // экранированного варианта нет — искать нечего
	}
	return extractTitleDataVariant(strings.ReplaceAll(page, `\"`, `"`))
}

// extractTitleDataVariant разбирает один вариант страницы (см. выше).
func extractTitleDataVariant(page string) (title, alt, status string) {
	loc := reTitleDir.FindStringSubmatchIndex(page)
	if loc == nil {
		return "", "", ""
	}
	dir := page[loc[2]:loc[3]]

	// Окно вокруг якоря: основной объект начинается за ~2 КБ до
	// translate_status (id, названия, тип, возрастной лимит, статус)
	// и продолжается после него (branches, счётчики).
	windowStart := loc[0] - 2500
	if windowStart < 0 {
		windowStart = 0
	}
	windowEnd := loc[1] + 500
	if windowEnd > len(page) {
		windowEnd = len(page)
	}
	window := page[windowStart:windowEnd]

	// Страховка: окно должно принадлежать нашему тайтлу (dir встречается
	// в payload и у карточек рекомендаций).
	if !strings.Contains(window, `"dir":"`+dir+`"`) {
		return "", "", ""
	}

	title = parsers.CleanTitle(firstMatch(reMainName, window))
	secondary := parsers.CleanTitle(firstMatch(reSecName, window))
	another := parsers.CleanTitle(firstMatch(reAltName, window))

	var statusID int
	var statusName string
	if m := reStatus.FindStringSubmatch(window); m != nil {
		statusID, _ = strconv.Atoi(m[1])
		statusName = m[2]
	}
	status = mapStatus(statusName, statusID)

	return title, joinAltTitles(secondary, another, title), status
}

// firstMatch возвращает первую группу первого совпадения ("" — нет).
func firstMatch(re *regexp.Regexp, s string) string {
	m := re.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	return m[1]
}

// mapStatus переводит статус движка Remanga в publication_status манги.
// Приоритет — id (стабильный enum), имя — страховка на случай новых id.
// Замеченные значения: 1 Закончен, 2 Продолжается, 3 Заморожен, 4 Анонс.
func mapStatus(name string, id int) string {
	switch id {
	case 1:
		return "completed"
	case 2:
		return "ongoing"
	case 3:
		return "hiatus"
	case 4:
		return "" // анонс: публикации ещё нет
	}

	name = strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.Contains(name, "закончен"):
		return "completed"
	case strings.Contains(name, "продолжает"):
		return "ongoing"
	case strings.Contains(name, "заморожен"):
		return "hiatus"
	case strings.Contains(name, "заброшен"):
		return "discontinued"
	case strings.Contains(name, "анонс"):
		return ""
	default:
		return "" // неизвестное значение — статус не меняем
	}
}

// joinAltTitles собирает альтернативные названия из secondary и another,
// убирая дубли и совпадения с основным названием (title).
func joinAltTitles(secondary, another, title string) string {
	var result []string
	seen := make(map[string]bool)
	for _, part := range []string{secondary, another} {
		part = strings.TrimSpace(part)
		if part == "" || part == title || seen[part] {
			continue
		}
		seen[part] = true
		result = append(result, part)
	}
	return strings.Join(result, " / ")
}
