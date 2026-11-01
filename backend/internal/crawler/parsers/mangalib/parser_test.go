package mangalib

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers"
)

// Тестовый ответ повторяет реальный JSON GET api.imglib.info/api/manga/{id}--{slug}
// (заголовки Referer + Site-Id: 1, проверено 2026-11-01).
const mangalibCardFixture = `{"data":{"id":274927,"name":"Кот Баюн","rus_name":"Кот Баюн","eng_name":"The Cat Bayun","model":"manga","slug":"kot-baiun","slug_url":"274927--kot-baiun","status":{"id":2,"label":"Завершён"},"type":{"id":8,"label":"Руманга"}}}`

func TestMangalibCard(t *testing.T) {
	driver := NewParser()
	card, err := driver.ParseMangaCard([]byte(mangalibCardFixture))
	if err != nil {
		t.Fatalf("ошибка парсинга: %v", err)
	}

	if card.Title != "Кот Баюн" {
		t.Errorf("название: получено %q", card.Title)
	}
	if card.AltTitles != "The Cat Bayun" {
		t.Errorf("альт. название: получено %q", card.AltTitles)
	}
	if card.PublicationStatus != "completed" {
		t.Errorf("статус публикации: получено %q, ожидалось %q", card.PublicationStatus, "completed")
	}
	if len(card.Chapters) != 0 {
		t.Errorf("глав: получено %d, ожидалось 0 (список глав догружается из API)", len(card.Chapters))
	}
}

func TestMangalibCardBadBody(t *testing.T) {
	// Битый JSON или пустой ответ — ошибка (сигнал, что API-стратегия сломалась).
	driver := NewParser()
	for _, body := range []string{"", "<html>SPA-каркас</html>", `{"data":{}}`} {
		if _, err := driver.ParseMangaCard([]byte(body)); err == nil {
			t.Errorf("тело %q должно возвращать ошибку (нет названия)", body)
		}
	}
}

func TestMangalibAltEqualsTitle(t *testing.T) {
	// eng_name совпадает с названием — альтернативное не пишется.
	body := `{"data":{"name":"One Piece","rus_name":"One Piece","eng_name":"One Piece","status":{"label":"Онгоинг"}}}`
	driver := NewParser()
	card, err := driver.ParseMangaCard([]byte(body))
	if err != nil {
		t.Fatalf("ошибка парсинга: %v", err)
	}
	if card.AltTitles != "" {
		t.Errorf("альт. название: получено %q, ожидалось пусто", card.AltTitles)
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
		{"https://mangalib.me/ru/manga/277246--maou-taoshita-shi-kaeru-ka?ui=1", true},    // query отсекается
		{"https://mangalib.me/ru/manga/--no-id", false},                                   // нет числового id
		{"https://mangalib.me/ru/manga/277246--", false},                                  // нет слага
		{"https://mangalib.me/ru/manga/abc--slug", false},                                 // id не число
		{"https://mangalib.me/ru/manga", false},                                           // без карточки
		{"https://mangalib.me/ru/manga/", false},                                          // без карточки
		{"https://mangalib.me/ru/catalog", false},                                         // каталог
		{"https://mangalib.me/ru/277246--maou-taoshita-shi-kaeru-ka/read/v01/c01", false}, // читалка
		{"https://mangalib.me/ru/book/123--some-book", false},                             // ранобэ (в sitemap не попадают, но проверим)
		{"https://mangalib.me/en/manga/904--gingitsune", false},                           // чужая локаль
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

// CardRequest: адрес API-карточки по id--slug и обязательные заголовки
// (Referer — иначе 403, Site-Id — иначе 404).
func TestMangalibCardRequest(t *testing.T) {
	var driver parsers.CardRequestProvider = NewParser()

	rawURL, headers := driver.CardRequest("ru/manga/274927--kot-baiun")
	if rawURL != "https://api.imglib.info/api/manga/274927--kot-baiun" {
		t.Errorf("адрес карточки: %q", rawURL)
	}

	got := map[string]string{}
	for _, h := range headers {
		got[h.Key] = h.Value
	}
	if got["Referer"] != "https://mangalib.me/" {
		t.Errorf("Referer: %q, ожидалось %q", got["Referer"], "https://mangalib.me/")
	}
	if got["Site-Id"] != "1" {
		t.Errorf("Site-Id: %q, ожидалось %q", got["Site-Id"], "1")
	}

	// url-часть не карточки манги — пустой адрес (краулер не должен
	// получить мусорный запрос).
	if u, _ := driver.CardRequest("ru/book/123--some-book"); u != "" {
		t.Errorf("чужая url-часть: получен адрес %q, ожидался пустой", u)
	}
}

// Статусы — значения фильтра /ru/catalog?status[] (совпадают с label API).
func TestMangalibStatusFromLabel(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Онгоинг", "ongoing"},
		{"Завершён", "completed"},
		{"Заморожен", "hiatus"},
		{"Заброшен", "discontinued"},
		{"Анонс", ""},
		{"Новый статус", ""}, // неизвестное — не меняем
	}

	for _, tc := range cases {
		if got := statusFromLabel(tc.in); got != tc.want {
			t.Errorf("statusFromLabel(%q): получено %q, ожидалось %q", tc.in, got, tc.want)
		}
	}
}

// ——— тесты догрузки глав из API ———

func TestMangalibLoadChapters(t *testing.T) {
	driver := NewParser()

	var calls []string
	var gotHeaders []parsers.ExtraHeader
	load := func(ctx context.Context, rawURL string, headers ...parsers.ExtraHeader) ([]byte, error) {
		calls = append(calls, rawURL)
		gotHeaders = headers
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

	// API отвечает 403 без Referer и 404 без Site-Id — заголовки
	// обязаны доходить до загрузчика.
	headers := map[string]string{}
	for _, h := range gotHeaders {
		headers[h.Key] = h.Value
	}
	if headers["Referer"] != "https://mangalib.me/" {
		t.Errorf("Referer запроса к API: %q", headers["Referer"])
	}
	if headers["Site-Id"] != "1" {
		t.Errorf("Site-Id запроса к API: %q, ожидалось %q", headers["Site-Id"], "1")
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

	// idSlug: третий сегмент целиком, для адреса карточки в API.
	if got := idSlugFromURLPart("ru/manga/274927--kot-baiun"); got != "274927--kot-baiun" {
		t.Errorf("idSlugFromURLPart: получено %q", got)
	}
}
