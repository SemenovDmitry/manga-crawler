package flame

import (
	"strings"
	"testing"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers"
)

// Тестовый фрагмент повторяет реальную разметку flamecomics.xyz
// (см. probe-output/flamecomics.xyz/debug-series_2.html): Mantine-классы
// с хешами версии, h1 mantine-Title-root, статус в Badge, ссылки глав
// href="/series/<id>/<16 hex>" и __NEXT_DATA__ со структурой
// props.pageProps.series / props.pageProps.chapters.
const flameCardFixture = `
<!DOCTYPE html>
<html><head><title>30 Years Have Passed Since the Prologue - Flame Comics</title></head>
<body>
<div class="SeriesPage-module__zgaxnW__paper m_1b7284a3 mantine-Paper-root">
<h1 class="m_8a5d1357 mantine-Title-root" data-order="1">30 Years Have Passed Since the Prologue</h1>
<p class="mantine-focus-auto SeriesPage-module__zgaxnW__altTitles m_b6d8b162 mantine-Text-root">30 Years Since the Prologue</p>
<div class="m_347db0ec mantine-Badge-root" data-variant="outline"><span class="m_5add502a mantine-Badge-label">Ongoing</span></div>
<a href="/genre/Academy"><div class="m_347db0ec mantine-Badge-root"><span class="m_5add502a mantine-Badge-label">Academy</span></div></a>
<a href="/genre/Action"><div class="m_347db0ec mantine-Badge-root"><span class="m_5add502a mantine-Badge-label">Action</span></div></a>
<p class="mantine-focus-auto m_b6d8b162 mantine-Text-root" data-size="lg">12 Chapters</p>
<a class="ChapterCard-module__CWFuLG__chapterWrapper" href="/series/167/aaaabbbbccccdddd"><p>Chapter <!-- -->12<!-- --> - Last</p></a>
<a class="ChapterCard-module__CWFuLG__chapterWrapper" href="/series/167/1111222233334444"><p>Chapter <!-- -->11<!-- --> - Previous</p></a>
</div>
<script id="__NEXT_DATA__" type="application/json">{"props":{"pageProps":{
"series":{"series_id":167,"title":"30 Years Have Passed Since the Prologue",
"altTitles":["30 Years Since the Prologue","Пролог минул 30 лет назад","30 Years Since the Prologue"],
"status":"Ongoing","type":"Manhwa","year":2026},
"chapters":[
{"chapter_id":9,"series_id":167,"chapter":"12.00","title":"Last","release_date":1,"token":"aaaabbbbccccdddd"},
{"chapter_id":8,"series_id":167,"chapter":"11.50","title":"Interlude","release_date":2,"token":"1111222233334444"},
{"chapter_id":1,"series_id":167,"chapter":"0.00","title":"Prologue","release_date":3,"token":"eeeeffff00001111"}
]}}}</script>
</body></html>
`

func TestFlameCard(t *testing.T) {
	driver := NewParser()
	card, err := driver.ParseMangaCard([]byte(flameCardFixture))
	if err != nil {
		t.Fatalf("ошибка парсинга: %v", err)
	}

	if card.Title != "30 Years Have Passed Since the Prologue" {
		t.Errorf("название: получено %q, ожидалось %q", card.Title, "30 Years Have Passed Since the Prologue")
	}

	wantAlt := "30 Years Since the Prologue / Пролог минул 30 лет назад"
	if card.AltTitles != wantAlt {
		t.Errorf("альт. названия: получено %q, ожидалось %q", card.AltTitles, wantAlt)
	}

	if card.PublicationStatus != "ongoing" {
		t.Errorf("статус публикации: получено %q, ожидалось %q", card.PublicationStatus, "ongoing")
	}

	normalized := parsers.NormalizeChapters(card.Chapters)
	if len(normalized) != 3 {
		t.Fatalf("глав: получено %d, ожидалось 3", len(normalized))
	}
	// Номера нормализуются: "0.00" -> "0", "11.50" -> "11.5", "12.00" -> "12"
	if normalized[0].Number != "0" || normalized[1].Number != "11.5" || normalized[2].Number != "12" {
		t.Errorf("порядок глав: получено %s, %s, %s; ожидалось 0, 11.5, 12",
			normalized[0].Number, normalized[1].Number, normalized[2].Number)
	}
	if normalized[2].Title != "Last" {
		t.Errorf("подпись главы 12: получено %q, ожидалось %q", normalized[2].Title, "Last")
	}
}

func TestFlameCardFallbackMarkup(t *testing.T) {
	// Страница без __NEXT_DATA__: заголовок из h1 mantine-Title-root,
	// главы — по ссылкам href="/series/<id>/<hex>" без номера.
	page := strings.Replace(flameCardFixture, `<script id="__NEXT_DATA__" type="application/json">`, `<script id="OTHER">`, 1)
	// Вырезаем JSON: скрипт OTHER больше не __NEXT_DATA__.
	page = strings.Split(page, `<script id="OTHER">`)[0] + `</body></html>`

	driver := NewParser()
	card, err := driver.ParseMangaCard([]byte(page))
	if err != nil {
		t.Fatalf("ошибка парсинга: %v", err)
	}

	if card.Title != "30 Years Have Passed Since the Prologue" {
		t.Errorf("название: получено %q, ожидалось %q", card.Title, "30 Years Have Passed Since the Prologue")
	}
	if card.PublicationStatus != "" {
		t.Errorf("статус: получено %q, ожидалось пусто (в fallback не извлекается)", card.PublicationStatus)
	}

	// Три ссылки глав + дубли кнопок отсутствуют; 12-я глава встречается один раз.
	if len(card.Chapters) != 2 {
		t.Errorf("глав: получено %d, ожидалось 2", len(card.Chapters))
	}
}

func TestFlameCardNotACard(t *testing.T) {
	driver := NewParser()
	if _, err := driver.ParseMangaCard([]byte("<html><body>Error 404</body></html>")); err == nil {
		t.Error("страница без карточки должна возвращать ошибку")
	}
}

func TestFlameURLs(t *testing.T) {
	driver := NewParser()

	cases := []struct {
		url  string
		want bool
	}{
		{"https://flamecomics.xyz/series/2", true},
		{"https://flamecomics.xyz/series/167", true},
		{"https://flamecomics.xyz/series/2/0c9db8012fbd1257", false}, // страница главы
		{"https://flamecomics.xyz/series/2/364db6fd6bef182e", false},
		{"https://flamecomics.xyz/novel/8", false},          // новелла
		{"https://flamecomics.xyz/series/", false},          // без id
		{"https://flamecomics.xyz/series/abc", false},       // не число
		{"https://flamecomics.xyz/series/0", false},         // id не бывает нулём
		{"https://flamecomics.xyz/series/02", false},        // ведущий ноль — не канонический id
		{"https://flamecomics.xyz/series/2/cover.webp", false},
		{"https://flamecomics.xyz/browse", false},
		{"https://flamecomics.xyz/genre/Action", false},
		{"https://flamecomics.xyz/blog/1", false},
		{"https://flamecomics.xyz/", false},
		{"https://1.seimanga.me/asura_2024", false}, // чужой движок
	}

	for _, tc := range cases {
		if got := driver.IsMangaURL(tc.url); got != tc.want {
			t.Errorf("IsMangaURL(%s): получено %v, ожидалось %v", tc.url, got, tc.want)
		}
	}

	part, err := driver.MangaURLPart("https://flamecomics.xyz/series/2/")
	if err != nil {
		t.Fatalf("MangaURLPart: %v", err)
	}
	if part != "series/2" {
		t.Errorf("MangaURLPart: получено %q, ожидалось %q", part, "series/2")
	}
}

// Статусы — значения поля status в __NEXT_DATA__ (все встречены в /browse).
func TestFlameMapStatus(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Ongoing", "ongoing"},
		{"ongoing", "ongoing"},
		{"Completed", "completed"},
		{"Hiatus", "hiatus"},
		{"Dropped", "discontinued"},
		{"Cancelled", "cancelled"},
		{"Coming Soon", ""}, // ещё не публиковалась — статус не меняем
		{"", ""},
	}

	for _, tc := range cases {
		if got := mapStatus(tc.in); got != tc.want {
			t.Errorf("mapStatus(%q): получено %q, ожидалось %q", tc.in, got, tc.want)
		}
	}
}

// Нормализация номеров: "311.00" -> "311" (иначе сравнение глав в БД
// увидит "311" и "311.00" разными главами).
func TestFlameNormalizeChapterNumber(t *testing.T) {
	cases := []struct{ in, want string }{
		{"311.00", "311"},
		{"11.50", "11.5"},
		{"0.00", "0"},
		{"12", "12"},
		{"", ""},
	}

	for _, tc := range cases {
		if got := normalizeChapterNumber(tc.in); got != tc.want {
			t.Errorf("normalizeChapterNumber(%q): получено %q, ожидалось %q", tc.in, got, tc.want)
		}
	}
}
