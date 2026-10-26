package mangalib

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers"
)

// Тестовый фрагмент повторяет реальный SSR-ответ карточки с роботным
// User-Agent (см. .tmp-googlebot/card_googlebot.html): JSON-LD
// (BreadcrumbList + CreativeWorkSeries) и текстовый инфоблок с метками
// «Статус»/«Глав». Классы-хеши bel_* в фикстуре сохранены как в дампе,
// но парсер на них не завязан.
const mangalibCardFixture = `
<!DOCTYPE html><html lang="ru"><head><title>Я победил Владыку демонов, пожалуй, пора домой / Maou taoshita shi kaeru ka читать мангу онлайн • MangaLIB</title>
<script type="application/ld+json">[{"@context":"https://schema.org","@type":"BreadcrumbList","itemListElement":[{"@type":"ListItem","position":1,"name":"Каталог","item":"https://mangalib.me/ru/catalog"},{"@type":"ListItem","position":3,"name":"Я победил Владыку демонов, пожалуй, пора домой","item":"https://mangalib.me/ru/manga/277246--maou-taoshita-shi-kaeru-ka"}]},{"@context":"https://schema.org","@type":"CreativeWorkSeries","url":"https://mangalib.me/ru/manga/277246--maou-taoshita-shi-kaeru-ka","name":"Я победил Владыку демонов, пожалуй, пора домой","headline":"Я победил Владыку демонов, пожалуй, пора домой","alternativeHeadline":["«魔王倒したし帰るか»","Maou taoshita shi kaeru ka","I've Defeated the Demon Lord, so I Guess I'll Head Home"],"image":"https://cover.imglib.info/uploads/cover/maou-taoshita-shi-kaeru-ka/cover/65229be8.jpg","identifier":"Манга","isFamilyFriendly":false,"inLanguage":"Russian","typicalAgeRange":"18+","copyrightYear":2026}]</script>
</head><body>
<h1 class="avz_av1"><!----><span>Я победил Владыку демонов, пожалуй, пора домой</span></h1>
<div class="bel_y"><div class="bel_ca">Выпуск</div><div class="bel_cu"><span>2026 г.</span></div></div>
<div class="bel_y"><div class="bel_ca">Страна</div><div class="bel_cu"><span>—</span></div></div>
<div class="bel_y"><div class="bel_ca">Глав</div><div class="bel_cu"><span>2</span></div></div>
<a href="/ru/catalog?status%5B%5D=1" class="bel_y"><div class="bel_ca">Статус</div><div class="bel_cu"><span>Онгоинг</span></div></a>
<a href="/ru/catalog?scanlate_status%5B%5D=1" class="bel_y"><div class="bel_ca">Перевод</div><div class="bel_cu"><span>Продолжается</span></div></a>
</body></html>
`

func TestMangalibCard(t *testing.T) {
	driver := NewParser()
	card, err := driver.ParseMangaCard([]byte(mangalibCardFixture))
	if err != nil {
		t.Fatalf("ошибка парсинга: %v", err)
	}

	if card.Title != "Я победил Владыку демонов, пожалуй, пора домой" {
		t.Errorf("название: получено %q", card.Title)
	}

	wantAlt := "«魔王倒したし帰るか» / Maou taoshita shi kaeru ka / I've Defeated the Demon Lord, so I Guess I'll Head Home"
	if card.AltTitles != wantAlt {
		t.Errorf("альт. названия:\n  получено %q\n  ожидалось %q", card.AltTitles, wantAlt)
	}

	if card.PublicationStatus != "ongoing" {
		t.Errorf("статус публикации: получено %q, ожидалось %q", card.PublicationStatus, "ongoing")
	}

	if len(card.Chapters) != 0 {
		t.Errorf("глав: получено %d, ожидалось 0 (список глав догружается из API)", len(card.Chapters))
	}
}

func TestMangalibCardSPAStub(t *testing.T) {
	// SPA-каркас (8.4 КБ), который сайт отдаёт браузерному UA:
	// ни JSON-LD, ни h1 — страница должна уйти в bad-page дамп.
	spaStub := `<!DOCTYPE html><html lang="ru" data-id="1"><head><meta charset="utf-8"></head><body><div id="app"></div><script type="module" src="https://mangalib.me/build/assets/main.js"></script></body></html>`

	driver := NewParser()
	if _, err := driver.ParseMangaCard([]byte(spaStub)); err == nil {
		t.Error("SPA-каркас должен возвращать ошибку (сигнал, что UA-стратегия сломалась)")
	}
}

func TestMangalibCardFallbackH1(t *testing.T) {
	// JSON-LD сменил формат — карточка собирается из h1 SSR-разметки.
	page := `<html><body><h1 class="avz_av1"><span>Две вишни</span></h1></body></html>`

	driver := NewParser()
	card, err := driver.ParseMangaCard([]byte(page))
	if err != nil {
		t.Fatalf("ошибка парсинга: %v", err)
	}
	if card.Title != "Две вишни" {
		t.Errorf("название: получено %q, ожидалось %q", card.Title, "Две вишни")
	}
	if card.PublicationStatus != "" {
		t.Errorf("статус: получено %q, ожидалось пусто", card.PublicationStatus)
	}
}

func TestMangalibCardNotACard(t *testing.T) {
	driver := NewParser()
	if _, err := driver.ParseMangaCard([]byte("<html><body>Error 404</body></html>")); err == nil {
		t.Error("страница без карточки должна возвращать ошибку")
	}
}

func TestMangalibURLs(t *testing.T) {
	driver := NewParser()

	cases := []struct {
		url  string
		want bool
	}{
		{"https://mangalib.me/ru/manga/277246--maou-taoshita-shi-kaeru-ka", true},
		{"https://mangalib.me/ru/manga/175317--futari-de-cherry", true},
		{"https://mangalib.me/ru/manga/175317--futari-de-cherry/", true}, // трейлинг-слэш
		{"https://mangalib.me/ru/manga/904--gingitsune", true},
		{"https://mangalib.me/ru/manga/277246--maou-taoshita-shi-kaeru-ka?ui=1", true}, // query отсекается
		{"https://mangalib.me/ru/manga/--no-id", false},                                // нет числового id
		{"https://mangalib.me/ru/manga/277246--", false},                               // нет слага
		{"https://mangalib.me/ru/manga/abc--slug", false},                              // id не число
		{"https://mangalib.me/ru/manga", false},                                        // без карточки
		{"https://mangalib.me/ru/manga/", false},                                       // без карточки
		{"https://mangalib.me/ru/catalog", false},                                      // каталог
		{"https://mangalib.me/ru/277246--maou-taoshita-shi-kaeru-ka/read/v01/c01", false}, // читалка
		{"https://mangalib.me/ru/book/123--some-book", false},                          // ранобэ (в sitemap не попадают, но проверим)
		{"https://mangalib.me/en/manga/904--gingitsune", false},                        // чужая локаль
		{"https://mangalib.me/", false},
		{"https://1.seimanga.me/asura_2024", false}, // чужой движок
	}

	for _, tc := range cases {
		if got := driver.IsMangaURL(tc.url); got != tc.want {
			t.Errorf("IsMangaURL(%s): получено %v, ожидалось %v", tc.url, got, tc.want)
		}
	}

	for _, tc := range []struct{ url, want string }{
		{"https://mangalib.me/ru/manga/277246--maou-taoshita-shi-kaeru-ka", "ru/manga/277246--maou-taoshita-shi-kaeru-ka"},
		{"https://mangalib.me/ru/manga/175317--futari-de-cherry/", "ru/manga/175317--futari-de-cherry"},
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

// Роботный UA: сайт отдаёт SSR только роботам (проверено probe).
func TestMangalibCardUserAgent(t *testing.T) {
	var driver parsers.SourceDriver = NewParser()
	ua, ok := driver.(parsers.CardUserAgent)
	if !ok {
		t.Fatal("драйвер mangalib должен реализовывать parsers.CardUserAgent")
	}
	if !strings.Contains(ua.CardUserAgent(), "Googlebot") {
		t.Errorf("CardUserAgent: получено %q, ожидался роботный UA", ua.CardUserAgent())
	}
}

// Статусы — значения фильтра /ru/catalog?status[] (проверено на карточках).
func TestMangalibExtractStatus(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Онгоинг", "ongoing"},
		{"Завершён", "completed"},
		{"Заморожен", "hiatus"},
		{"Заброшен", "discontinued"},
		{"Анонс", ""},
		{"Новый статус", ""}, // неизвестное — не меняем
	}

	for _, tc := range cases {
		page := `<div class="x">Статус</div><div class="y"><span>` + tc.in + `</span></div>`
		if got := extractStatus(page); got != tc.want {
			t.Errorf("extractStatus(%q): получено %q, ожидалось %q", tc.in, got, tc.want)
		}
	}
}

// ——— тесты догрузки глав из API ———

func TestMangalibLoadChapters(t *testing.T) {
	driver := NewParser()

	var calls []string
	load := func(ctx context.Context, rawURL string) ([]byte, error) {
		calls = append(calls, rawURL)
		if !strings.HasSuffix(rawURL, "/api/manga/maou-taoshita-shi-kaeru-ka/chapters") {
			t.Errorf("неожиданный запрос API: %s", rawURL)
		}
		resp := apiChapters{}
		resp.Data = []struct {
			Volume string `json:"volume"`
			Number string `json:"number"`
			Name   string `json:"name"`
		}{
			{Volume: "1", Number: "1", Name: "Герой."},
			{Volume: "1", Number: "2", Name: "Воин."},
		}
		return json.Marshal(resp)
	}

	card := &parsers.MangaCard{Title: "Я победил Владыку демонов"}
	if err := driver.LoadChapters(context.Background(), load, "ru/manga/277246--maou-taoshita-shi-kaeru-ka", card); err != nil {
		t.Fatalf("догрузка глав: %v", err)
	}

	if len(card.Chapters) != 2 {
		t.Fatalf("глав: получено %d, ожидалось 2", len(card.Chapters))
	}
	normalized := parsers.NormalizeChapters(card.Chapters)
	if normalized[0].Number != "1" || normalized[1].Number != "2" {
		t.Errorf("порядок глав: %s, %s; ожидалось 1, 2", normalized[0].Number, normalized[1].Number)
	}
	if normalized[0].Volume != "1" || normalized[0].Title != "Герой." {
		t.Errorf("глава 1: том=%q подпись=%q", normalized[0].Volume, normalized[0].Title)
	}
	if len(calls) != 1 {
		t.Errorf("запросов к API: %d, ожидался 1", len(calls))
	}
}

func TestMangalibSlugFromURLPart(t *testing.T) {
	cases := []struct{ in, want string }{
		{"ru/manga/277246--maou-taoshita-shi-kaeru-ka", "maou-taoshita-shi-kaeru-ka"},
		{"ru/manga/904--gingitsune", "gingitsune"},
		{"manga/904--gingitsune", ""},  // без локали
		{"ru/manga/904--", ""},         // пустой слаг
		{"ru/manga/gingitsune", ""},    // без id
		{"ru/manga/904--a--b", "a--b"}, // slug с дефисами-разделителями
	}
	for _, tc := range cases {
		if got := slugFromURLPart(tc.in); got != tc.want {
			t.Errorf("slugFromURLPart(%q): получено %q, ожидалось %q", tc.in, got, tc.want)
		}
	}
}
