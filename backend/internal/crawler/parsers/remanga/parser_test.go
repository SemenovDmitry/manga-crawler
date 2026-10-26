package remanga

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers"
)

// Тестовый фрагмент повторяет реальный RSC-payload страницы тайтла
// (см. probe-output/remanga.org/debug-manga_pmd-white-rose_chapters.html):
// react-query кэш с основным объектом тайтла (якорь translate_status),
// карточкой рекомендации (без translate_status) и HTML-обёрткой с h1.
// Кавычки в payload — чистые (в сыром HTML Next.js не экранирует их
// повсеместно, проверено на реальных дампах).
const remangaCardFixture = `
<!DOCTYPE html><html lang="ru-RU"><head><title>PMD: White Rose</title></head><body>
<h1 class="cs-text text-2xl leading-2xl font-bold text-foreground cs-layout-title-text leading-lg">ПМД: Белая Роза</h1>
<script>self.__next_f.push([1,"40:{"dehydratedAt":1792957094943,"state":{"data":{"json":{"meta":{},"id":154413,"secondary_name":"PMD: White Rose","main_name":"ПМД: Белая Роза","another_name":"","type":{"id":4,"name":"Западный комикс"},"age_limit":{"id":1,"name":"16+"},"status":{"id":2,"name":"Продолжается"},"translate_status":{"id":2,"name":"Продолжается"},"dir":"pmd-white-rose","description":"История о вселенной Pokemon Mystery Dungeon","issue_year":2019,"avg_rating":"7.1","cover":{"low":"/media/titles/pmd-white-rose/cover.webp"},"count_chapters":179,"fresh_rating":"7.11","first_chapter":{"id":1854885,"tome":1,"chapter":"0"},"bookmark_type":null,"branches":[{"id":154342,"img":{},"subscribed":false,"total_votes":7027,"count_chapters":179,"publishers":[{"id":15068,"name":"27th Team"}]}]}}}"])</script>
<script>self.__next_f.push([1,"41:{"state":{"data":{"json":[{"id":99001,"secondary_name":"Another Title","main_name":"Другой тайтл","type":{"id":1,"name":"Манга"},"status":{"id":2,"name":"Продолжается"},"dir":"another-title"}]}}}"])</script>
</body></html>
`

func TestRemangaCard(t *testing.T) {
	driver := NewParser()
	card, err := driver.ParseMangaCard([]byte(remangaCardFixture))
	if err != nil {
		t.Fatalf("ошибка парсинга: %v", err)
	}

	if card.Title != "ПМД: Белая Роза" {
		t.Errorf("название: получено %q, ожидалось %q", card.Title, "ПМД: Белая Роза")
	}

	// secondary_name совпадает с... нет, отличается от main_name и попадает
	// в альтернативные; another_name пустой.
	wantAlt := "PMD: White Rose"
	if card.AltTitles != wantAlt {
		t.Errorf("альт. названия: получено %q, ожидалось %q", card.AltTitles, wantAlt)
	}

	if card.PublicationStatus != "ongoing" {
		t.Errorf("статус публикации: получено %q, ожидалось %q", card.PublicationStatus, "ongoing")
	}

	if len(card.Chapters) != 0 {
		t.Errorf("глав: получено %d, ожидалось 0 (список глав догружается из API)", len(card.Chapters))
	}
}

func TestRemangaCardFallbackH1(t *testing.T) {
	// Payload не распознался (изменился формат) — карточка собирается
	// из h1 разметки: без альтернативок и статуса, но не в bad-page.
	page := `<html><body><h1 class="cs-text text-2xl cs-layout-title-text">Лимб</h1></body></html>`

	driver := NewParser()
	card, err := driver.ParseMangaCard([]byte(page))
	if err != nil {
		t.Fatalf("ошибка парсинга: %v", err)
	}

	if card.Title != "Лимб" {
		t.Errorf("название: получено %q, ожидалось %q", card.Title, "Лимб")
	}
	if card.PublicationStatus != "" {
		t.Errorf("статус: получено %q, ожидалось пусто", card.PublicationStatus)
	}
}

func TestRemangaCardNotACard(t *testing.T) {
	driver := NewParser()
	if _, err := driver.ParseMangaCard([]byte("<html><body>Error 404</body></html>")); err == nil {
		t.Error("страница без карточки должна возвращать ошибку (уходит в bad-page дамп)")
	}
}

// Payload в hydration-скрипте экранирован (\"ключ\"): страница должна
// распарситься так же, как с чистыми кавычками (реальный дамп содержит
// оба представления).
func TestRemangaCardEscapedPayload(t *testing.T) {
	escaped := `{"state":{"data":{"json":{"meta":{},"id":154413,\"secondary_name\":\"PMD: White Rose\",\"main_name\":\"ПМД: Белая Роза\",\"another_name\":\"\",\"type\":{\"id\":4,\"name\":\"Западный комикс\"},\"age_limit\":{\"id\":1,\"name\":\"16+\"},\"status\":{\"id\":2,\"name\":\"Продолжается\"},\"translate_status\":{\"id\":2,\"name\":\"Продолжается\"},\"dir\":\"pmd-white-rose\",\"branches\":[{\"id\":154342}]}}}}`
	page := `<html><body><script>self.__next_f.push([1,"40:` + escaped + `"])</script></body></html>`

	driver := NewParser()
	card, err := driver.ParseMangaCard([]byte(page))
	if err != nil {
		t.Fatalf("ошибка парсинга: %v", err)
	}

	if card.Title != "ПМД: Белая Роза" {
		t.Errorf("название: получено %q, ожидалось %q", card.Title, "ПМД: Белая Роза")
	}
	if card.AltTitles != "PMD: White Rose" {
		t.Errorf("альт. названия: получено %q, ожидалось %q", card.AltTitles, "PMD: White Rose")
	}
	if card.PublicationStatus != "ongoing" {
		t.Errorf("статус публикации: получено %q, ожидалось %q", card.PublicationStatus, "ongoing")
	}
}

func TestRemangaURLs(t *testing.T) {
	driver := NewParser()

	cases := []struct {
		url  string
		want bool
	}{
		{"https://remanga.org/manga/pmd-white-rose", true},                          // прямая карточка
		{"https://remanga.org/manga/pmd-white-rose/main", true},                     // подстраница из sitemap
		{"https://remanga.org/manga/pmd-white-rose/chapters", true},                 // подстраница из sitemap
		{"https://remanga.org/manga/pmd-white-rose/similar", true},                  // подстраница из sitemap
		{"https://remanga.org/manga/pmd-white-rose/cards", true},                    // подстраница из sitemap
		{"https://remanga.org/manga/pmd-white-rose/characters", true},               // подстраница из sitemap
		{"https://remanga.org/manga/pmd-white-rose/moments", true},                  // подстраница из sitemap
		{"https://remanga.org/manga/pmd-white-rose/posts", true},                    // подстраница из sitemap
		{"https://remanga.org/manga/pmd-white-rose/1854885", false},                 // страница главы
		{"https://remanga.org/manga/pmd-white-rose/1854885?page=1", false},          // глава с query
		{"https://remanga.org/manga/top", false},                                    // служебная
		{"https://remanga.org/manga/hot-updates", false},                            // служебная
		{"https://remanga.org/manga/pmd-white-rose/unknown-subpage", false},         // неизвестная подстраница
		{"https://remanga.org/manga/", false},                                       // без слага
		{"https://remanga.org/catalog", false},                                      // каталог
		{"https://remanga.org/", false},                                             // корень
		{"https://1.seimanga.me/asura_2024", false},                                 // чужой движок
	}

	for _, tc := range cases {
		if got := driver.IsMangaURL(tc.url); got != tc.want {
			t.Errorf("IsMangaURL(%s): получено %v, ожидалось %v", tc.url, got, tc.want)
		}
	}

	for _, tc := range []struct{ url, want string }{
		{"https://remanga.org/manga/pmd-white-rose", "manga/pmd-white-rose"},
		{"https://remanga.org/manga/pmd-white-rose/main", "manga/pmd-white-rose/main"},
		{"https://remanga.org/manga/pmd-white-rose/chapters", "manga/pmd-white-rose/chapters"},
	} {
		part, err := driver.MangaURLPart(tc.url)
		if err != nil {
			t.Fatalf("MangaURLPart(%s): %v", tc.url, err)
		}
		if part != tc.want {
			t.Errorf("MangaURLPart(%s): получено %q, ожидалось %q", tc.url, part, tc.want)
		}
	}
}

// Статусы — enum движка remanga (id из payload; имена — страховка).
func TestRemangaMapStatus(t *testing.T) {
	cases := []struct {
		name string
		id   int
		want string
	}{
		{"Закончен", 1, "completed"},
		{"Продолжается", 2, "ongoing"},
		{"Заморожен", 3, "hiatus"},
		{"Анонс", 4, ""},
		{"Неизвестный", 99, ""}, // неизвестный id — по имени
		{"Заброшен", 0, "discontinued"},
		{"", 0, ""},
	}

	for _, tc := range cases {
		if got := mapStatus(tc.name, tc.id); got != tc.want {
			t.Errorf("mapStatus(%q, %d): получено %q, ожидалось %q", tc.name, tc.id, got, tc.want)
		}
	}
}

// ——— тесты догрузки глав ———

// fakeLoader подменяет PageLoader: отвечает на запросы API заготовками.
type fakeLoader struct {
	calls []string
}

func (f *fakeLoader) load(ctx context.Context, rawURL string) ([]byte, error) {
	f.calls = append(f.calls, rawURL)

	switch {
	case strings.Contains(rawURL, "/api/titles/pmd-white-rose/"):
		card := apiTitleCard{}
		card.Content.Branches = []struct {
			ID            int `json:"id"`
			CountChapters int `json:"count_chapters"`
		}{{ID: 154342, CountChapters: 179}}
		return json.Marshal(card)

	case strings.Contains(rawURL, "branch_id=154342"):
		page := pageOf(rawURL)
		if page > 2 {
			return []byte(`{"content":[]}`), nil // конец списка
		}
		// Страница 2 — старые главы (index 1,2), страница 1 — новые (4,3).
		resp := apiChaptersPage{}
		if page == 1 {
			resp.Content = []apiChapterRaw{
				{Index: 4, Tome: 1, Chapter: "2.5", Name: "Пятая с половиной"},
				{Index: 3, Tome: 1, Chapter: "2", Name: "Четвёртая"},
			}
		} else {
			resp.Content = []apiChapterRaw{
				{Index: 2, Tome: 1, Chapter: "1", Name: ""},
				{Index: 1, Tome: 0, Chapter: "0", Name: "Пролог"},
			}
		}
		return json.Marshal(resp)

	default:
		return nil, errors.New("неожиданный запрос: " + rawURL)
	}
}

// apiChapterRaw — псевдоним структуры главы для тестовых заготовок
// (поле Content в apiChaptersPage имеет анонимный тип).
type apiChapterRaw = struct {
	ID      int    `json:"id"`
	Index   int    `json:"index"`
	Tome    int    `json:"tome"`
	Chapter string `json:"chapter"`
	Name    string `json:"name"`
}

func pageOf(rawURL string) int {
	_, after, _ := strings.Cut(rawURL, "page=")
	n := 0
	for _, r := range after {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func TestRemangaLoadChapters(t *testing.T) {
	driver := NewParser()
	fake := &fakeLoader{}
	card := &parsers.MangaCard{Title: "ПМД: Белая Роза"}

	err := driver.LoadChapters(context.Background(), fake.load, "manga/pmd-white-rose", card)
	if err != nil {
		t.Fatalf("догрузка глав: %v", err)
	}

	// 2 страницы по 2 главы, отсортированные NormalizeChapters по номеру.
	normalized := parsers.NormalizeChapters(card.Chapters)
	if len(normalized) != 4 {
		t.Fatalf("глав: получено %d, ожидалось 4", len(normalized))
	}
	if normalized[0].Number != "1" || normalized[3].Number != "4" {
		t.Errorf("порядок глав: %s..%s, ожидалось 1..4", normalized[0].Number, normalized[3].Number)
	}
	if normalized[3].Title != "Пятая с половиной" {
		t.Errorf("подпись главы 4: получено %q", normalized[3].Title)
	}
	if normalized[1].Volume != "1" {
		t.Errorf("том главы 2: получено %q, ожидалось 1", normalized[1].Volume)
	}
	if normalized[0].Volume != "" {
		t.Errorf("том главы 1 (tome=0): получено %q, ожидалось пусто", normalized[0].Volume)
	}

	// Запросы: карточка API, страницы глав 1 и 2, страница 3 (пустая,
	// подтверждает конец списка — 4 запроса).
	if len(fake.calls) != 4 {
		t.Errorf("запросов к API: %d, ожидалось 4 (%v)", len(fake.calls), fake.calls)
	}
}

func TestRemangaLoadChaptersNoBranches(t *testing.T) {
	driver := NewParser()
	empty := &fakeLoader{}
	card := &parsers.MangaCard{Title: "Тайтл без веток"}

	// Тайтл без веток: подменяем ответ карточки API.
	loader := func(ctx context.Context, rawURL string) ([]byte, error) {
		return []byte(`{"content":{"branches":[]}}`), nil
	}
	_ = empty

	err := driver.LoadChapters(context.Background(), loader, "manga/some-title", card)
	if err == nil {
		t.Error("тайтл без веток должен давать ошибку догрузки (карточка сохраняется, без глав)")
	}
	if len(card.Chapters) != 0 {
		t.Errorf("глав не должно появиться, получено %d", len(card.Chapters))
	}
}

func TestRemangaDirFromURLPart(t *testing.T) {
	cases := []struct{ in, want string }{
		{"manga/pmd-white-rose", "pmd-white-rose"},
		{"manga/limb", "limb"},
		{"asura_2024", ""},    // чужой формат
		{"manga/", ""},        // пустой слаг
		{"manga/a/b", ""},     // лишняя вложенность
	}
	for _, tc := range cases {
		if got := dirFromURLPart(tc.in); got != tc.want {
			t.Errorf("dirFromURLPart(%q): получено %q, ожидалось %q", tc.in, got, tc.want)
		}
	}
}
