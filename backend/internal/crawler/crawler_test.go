package crawler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers/grouple"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers/mgeko"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

func TestNeedsFullCrawl(t *testing.T) {
	scheduled := func(d time.Duration) sql.NullTime {
		return sql.NullTime{Time: time.Now().Add(d), Valid: true}
	}

	cases := []struct {
		name     string
		force    bool
		next     sql.NullTime
		wantFull bool
	}{
		{"расписание не назначено (ни разу не обходился)", false, sql.NullTime{}, true},
		{"следующий обход в будущем", false, scheduled(7 * 24 * time.Hour), false},
		{"срок расписания наступил", false, scheduled(-time.Hour), true},
		{"форсируем полный обход", true, scheduled(7 * 24 * time.Hour), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := storage.Source{NextFullCrawlAt: tc.next}

			if got := needsFullCrawl(Config{ForceFull: tc.force}, src, time.Now()); got != tc.wantFull {
				t.Errorf("needsFullCrawl: получено %v, ожидалось %v", got, tc.wantFull)
			}
		})
	}
}

func TestDriverFor(t *testing.T) {
	cases := []struct {
		engine string
		want   string
	}{
		{"mgeko", "*mgeko.Parser"},
		{"Mgeko", "*mgeko.Parser"},
		{"grouple", "*grouple.Parser"},
		{" grouple ", "*grouple.Parser"},
	}

	for _, tc := range cases {
		driver, err := DriverFor(tc.engine)
		if err != nil {
			t.Errorf("DriverFor(%q): неожиданная ошибка: %v", tc.engine, err)
			continue
		}
		if got := typeName(driver); got != tc.want {
			t.Errorf("DriverFor(%q): получено %s, ожидалось %s", tc.engine, got, tc.want)
		}
	}

	// Неизвестный движок должен давать ошибку, а не тихо отдавать grouple
	for _, engine := range []string{"", "неизвестный", "mgeko.cc"} {
		driver, err := DriverFor(engine)
		if err == nil {
			t.Errorf("DriverFor(%q): ожидалась ошибка, получен драйвер %s", engine, typeName(driver))
		}
		if driver != nil {
			t.Errorf("DriverFor(%q): при ошибке драйвер должен быть nil", engine)
		}
	}
}

func typeName(v any) string {
	switch v.(type) {
	case *mgeko.Parser:
		return "*mgeko.Parser"
	case *grouple.Parser:
		return "*grouple.Parser"
	default:
		return "неизвестный драйвер"
	}
}

// Компиляторная проверка: оба драйвера удовлетворяют контракту краулера.
var (
	_ parsers.SourceDriver  = (*grouple.Parser)(nil)
	_ parsers.SourceDriver  = (*mgeko.Parser)(nil)
	_ parsers.CatalogLister = (*grouple.Parser)(nil)
	_ Notifier              = (*QueueNotifier)(nil)
)

// ——— фейки для тестов без сети и БД ———

// fakeDriver — драйвер-заглушка: карточка всегда парсится, url-часть —
// последний сегмент пути.
type fakeDriver struct{}

func (fakeDriver) ParseMangaCard([]byte) (*parsers.MangaCard, error) {
	return &parsers.MangaCard{
		Title:    "Тестовая манга",
		Chapters: []parsers.Chapter{{Number: "1"}, {Number: "2"}},
	}, nil
}

func (fakeDriver) IsMangaURL(rawURL string) bool { return strings.Contains(rawURL, "/manga/") }

func (fakeDriver) MangaURLPart(rawURL string) (string, error) {
	i := strings.LastIndex(rawURL, "/")
	if i < 0 {
		return "", fmt.Errorf("не удалось выделить url-часть из %q", rawURL)
	}
	return rawURL[i+1:], nil
}

// fakeRepo — мок хранилища. Неиспользуемые методы встроенного nil-интерфейса
// паникуют — тест упадёт громко, если сущность начнёт звать лишнее.
type fakeRepo struct {
	repo
	states  map[string]storage.MangaState
	upserts []string
}

func (f *fakeRepo) GetMangaState(_ int, urlPart string) (storage.MangaState, bool, error) {
	st, ok := f.states[urlPart]
	return st, ok, nil
}

func (f *fakeRepo) UpsertManga(_ int, urlPart string, _, _, _ string, _ storage.PublicationStatus, _ int) (int64, error) {
	f.upserts = append(f.upserts, urlPart)
	return int64(len(f.upserts)), nil
}

// ——— автоотключение источника при ошибках парсинга ———

// brokenDriver — драйвер, чей парсинг карточки всегда падает.
type brokenDriver struct {
	fakeDriver
}

func (brokenDriver) ParseMangaCard([]byte) (*parsers.MangaCard, error) {
	return nil, errors.New("страница не является карточкой манги (заголовок не найден)")
}

// disablingRepo — мок хранилища с записью отключений источников
// и сервисных сообщений.
type disablingRepo struct {
	repo
	disabledIDs []int
	serviceText []string
}

func (r *disablingRepo) GetMangaState(_ int, _ string) (storage.MangaState, bool, error) {
	return storage.MangaState{}, false, nil
}

func (r *disablingRepo) UpsertManga(_ int, urlPart string, _, _, _ string, _ storage.PublicationStatus, _ int) (int64, error) {
	return 1, nil
}

func (r *disablingRepo) SetSourceEnabled(id int, enabled bool) (storage.Source, error) {
	if enabled {
		return storage.Source{ID: id, Enabled: true}, nil
	}
	r.disabledIDs = append(r.disabledIDs, id)
	return storage.Source{ID: id, Enabled: false}, nil
}

func (r *disablingRepo) EnqueueServiceNotification(chatID int64, text string) error {
	r.serviceText = append(r.serviceText, fmt.Sprintf("chat=%d: %s", chatID, text))
	return nil
}

func newDisablingCrawler(t *testing.T, srvURL string, limit int) (*Crawler, *disablingRepo) {
	t.Helper()
	store := &disablingRepo{}
	c := New(store, Config{
		CardFailureLimit: limit,
		OwnerChatID:      "42",
	})
	return c, store
}

func TestProcessNewURLsDisablesSourceAfterThreshold(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html>битая карточка</html>"))
	}))
	defer srv.Close()

	c, store := newDisablingCrawler(t, srv.URL, 3)
	src := crawlTarget{source: storage.Source{ID: 7, Name: "broken", BaseURL: srv.URL}, driver: brokenDriver{}}

	urls := []string{
		srv.URL + "/manga/a",
		srv.URL + "/manga/b",
		srv.URL + "/manga/c",
		srv.URL + "/manga/d", // порог (3) достигнут на предыдущей — эта не парсится
		srv.URL + "/manga/e",
	}

	res, err := c.processNewURLs(context.Background(), src, "Тест", urls, urlSet{})

	// Прервано автоотключением, а не отменой.
	var disabledErr errSourceDisabled
	if !errors.As(err, &disabledErr) {
		t.Fatalf("ожидалась errSourceDisabled, получено: %v", err)
	}

	if res.processed != 0 {
		t.Errorf("processed=%d, ожидалось 0 (парсинг всегда падает)", res.processed)
	}
	if len(store.disabledIDs) != 1 || store.disabledIDs[0] != 7 {
		t.Errorf("отключённые источники: %v, ожидался [7]", store.disabledIDs)
	}
	if len(store.serviceText) != 1 {
		t.Fatalf("сервисных сообщений: %d, ожидалось 1: %v", len(store.serviceText), store.serviceText)
	}
	text := store.serviceText[0]
	for _, want := range []string{"chat=42", "broken", "3 раз подряд", "отключён"} {
		if !strings.Contains(text, want) {
			t.Errorf("в сервисном сообщении нет %q:\n%s", want, text)
		}
	}
}

// switchableDriver — драйвер, который падает на теле-маркере "FAIL"
// и успешно парсит всё остальное.
type switchableDriver struct {
	fakeDriver
}

func (d *switchableDriver) ParseMangaCard(body []byte) (*parsers.MangaCard, error) {
	if strings.Contains(string(body), "FAIL") {
		return nil, errors.New("страница не является карточкой манги (заголовок не найден)")
	}
	return d.fakeDriver.ParseMangaCard(body)
}

func TestProcessNewURLsSuccessResetsFailureStreak(t *testing.T) {
	// Серия ошибок сбрасывается успешной карточкой: разовые сбои среди
	// рабочих страниц источник не отключают.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "bad") {
			_, _ = w.Write([]byte("FAIL: битая карточка"))
			return
		}
		_, _ = w.Write([]byte("<html>хорошая карточка</html>"))
	}))
	defer srv.Close()

	store := &disablingRepo{}
	c := New(store, Config{CardFailureLimit: 3, OwnerChatID: "42"})
	src := crawlTarget{
		source: storage.Source{ID: 5, Name: "mixed", BaseURL: srv.URL},
		driver: &switchableDriver{},
	}

	urls := []string{
		srv.URL + "/manga/bad1", // серия 1
		srv.URL + "/manga/bad2", // серия 2
		srv.URL + "/manga/good", // успех — серия сброшена
		srv.URL + "/manga/bad3", // серия 1
		srv.URL + "/manga/bad4", // серия 2
		srv.URL + "/manga/bad5", // серия 3 — порог, отключение
		srv.URL + "/manga/after",// после отключения не парсится
	}

	_, err := c.processNewURLs(context.Background(), src, "Тест", urls, urlSet{})

	var disabledErr errSourceDisabled
	if !errors.As(err, &disabledErr) {
		t.Fatalf("ожидалась errSourceDisabled, получено: %v", err)
	}
	if len(store.disabledIDs) != 1 || store.disabledIDs[0] != 5 {
		t.Errorf("отключённые источники: %v, ожидался [5]", store.disabledIDs)
	}
	if len(store.serviceText) != 1 {
		t.Fatalf("сервисных сообщений: %d, ожидалось 1", len(store.serviceText))
	}
	if !strings.Contains(store.serviceText[0], "3 раз подряд") {
		t.Errorf("в сообщении нет серии из 3 ошибок:\n%s", store.serviceText[0])
	}
}

func TestProcessNewURLsAutoDisableDisabledByConfig(t *testing.T) {
	// CardFailureLimit=0 — автоотключение выключено: ошибки копятся,
	// но источник не отключается и обход продолжается.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html>битая карточка</html>"))
	}))
	defer srv.Close()

	c, store := newDisablingCrawler(t, srv.URL, 0)
	src := crawlTarget{source: storage.Source{ID: 9, Name: "nolimit", BaseURL: srv.URL}, driver: brokenDriver{}}

	urls := []string{srv.URL + "/manga/a", srv.URL + "/manga/b", srv.URL + "/manga/c", srv.URL + "/manga/d", srv.URL + "/manga/e"}
	res, err := c.processNewURLs(context.Background(), src, "Тест", urls, urlSet{})
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if res.processed != 0 || res.skipped != 0 {
		t.Errorf("processed=%d skipped=%d (все карточки падают)", res.processed, res.skipped)
	}
	if len(store.disabledIDs) != 0 {
		t.Errorf("при пороге 0 источник отключаться не должен: %v", store.disabledIDs)
	}
	if len(store.serviceText) != 0 {
		t.Errorf("сервисных сообщений быть не должно: %v", store.serviceText)
	}
}

func TestSourceDisabledMessage(t *testing.T) {
	text := formatSourceDisabledMessage(
		SourceInfo{Name: "MangaLib", BaseURL: "https://mangalib.me"},
		20,
		errors.New("страница не является карточкой манги mangalib (SPA-каркас)"),
	)

	for _, want := range []string{
		"🛑", "MangaLib", "20 раз подряд", "SPA-каркас", "Включить обратно",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("в сообщении нет %q:\n%s", want, text)
		}
	}
	// HTML в тексте ошибки экранируется.
	if strings.Contains(text, "<b>Источник") == false {
		t.Errorf("заголовок потерял разметку:\n%s", text)
	}
}

func TestQueueNotifierServiceMessageWithoutOwner(t *testing.T) {
	// ownerChatID=0 (TELEGRAM_CHAT_ID не задан): сообщение не ставится в
	// очередь и не паникует.
	store := &disablingRepo{}
	n := NewQueueNotifier(store, 0)
	n.ServiceMessage("тест")
	if len(store.serviceText) != 0 {
		t.Errorf("без чата хозяина сообщений быть не должно: %v", store.serviceText)
	}
}

// ——— processNewURLs ———

func TestProcessNewURLs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html>карточка</html>"))
	}))
	defer srv.Close()

	store := &fakeRepo{states: map[string]storage.MangaState{}}
	c := New(store, Config{})
	src := crawlTarget{source: storage.Source{ID: 1, Name: "test", BaseURL: srv.URL}, driver: fakeDriver{}}

	urls := []string{
		srv.URL + "/manga/known", // уже известна (предзагружена в seen)
		srv.URL + "/manga/a",
		srv.URL + "/manga/a", // дубликат в самом списке
		"bad-url",            // url-часть не распознаётся
		srv.URL + "/manga/b",
	}

	seen := urlSet{"known": true}
	res, err := c.processNewURLs(context.Background(), src, "Тест", urls, seen)
	if err != nil {
		t.Fatalf("processNewURLs: неожиданная ошибка: %v", err)
	}
	if res.processed != 2 || res.skipped != 3 {
		t.Errorf("processed=%d skipped=%d, ожидалось 2/3", res.processed, res.skipped)
	}
	if len(store.upserts) != 2 {
		t.Errorf("сохранено карточек %d, ожидалось 2: %v", len(store.upserts), store.upserts)
	}

	// Повторный вызов с тем же seen: всё уже учтено, ничего не парсится.
	res2, err := c.processNewURLs(context.Background(), src, "Тест", urls, seen)
	if err != nil {
		t.Fatalf("processNewURLs (повтор): неожиданная ошибка: %v", err)
	}
	if res2.processed != 0 || res2.skipped != len(urls) {
		t.Errorf("повтор: processed=%d skipped=%d, ожидалось 0/%d", res2.processed, res2.skipped, len(urls))
	}
}

func TestProcessNewURLsStopsOnContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html>карточка</html>"))
	}))
	defer srv.Close()

	c := New(&fakeRepo{states: map[string]storage.MangaState{}}, Config{RequestDelay: time.Hour})
	src := crawlTarget{source: storage.Source{ID: 1, Name: "test", BaseURL: srv.URL}, driver: fakeDriver{}}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // отмена ещё до первого запроса

	res, err := c.processNewURLs(ctx, src, "Тест", []string{srv.URL + "/manga/a"}, urlSet{})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("ожидалась context.Canceled, получено %v", err)
	}
	if res.processed != 0 {
		t.Errorf("при отмене ничего не должно обрабатываться, processed=%d", res.processed)
	}
}

// ——— urlLister ———

func TestSitemapLister(t *testing.T) {
	mangaURL := "http://source-site/manga/a"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sitemap.xml" {
			fmt.Fprintf(w,
				`<?xml version="1.0" encoding="UTF-8"?>
				<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
					<url><loc>%s</loc></url>
					<url><loc>http://source-site/static/page</loc></url>
				</urlset>`, mangaURL)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	src := crawlTarget{source: storage.Source{Name: "test", BaseURL: srv.URL}, driver: fakeDriver{}}

	urls, err := sitemapLister{fetch: NewFetcher()}.list(context.Background(), src, 0)
	if err != nil {
		t.Fatalf("sitemapLister.list: неожиданная ошибка: %v", err)
	}
	if len(urls) != 1 || urls[0] != mangaURL {
		t.Errorf("получено %v, ожидалось [%s] — посторонние URL не отфильтрованы", urls, mangaURL)
	}
}

// fakeCatalogDriver — драйвер с каталогом из двух страниц.
type fakeCatalogDriver struct {
	fakeDriver
}

func (fakeCatalogDriver) CatalogPageURL(baseURL string, offset int) string {
	return fmt.Sprintf("%s/catalog?offset=%d", baseURL, offset)
}

func (fakeCatalogDriver) CatalogPageSize() int { return 2 }

func (d fakeCatalogDriver) MangaURLsFromCatalog(body []byte, baseURL string) []string {
	switch strings.TrimSpace(string(body)) {
	case "page-0":
		return []string{baseURL + "/manga/a", baseURL + "/manga/b"}
	case "page-1":
		return []string{baseURL + "/manga/c"}
	default:
		return nil // каталог кончился
	}
}

func TestCatalogLister(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("offset") {
		case "0":
			_, _ = w.Write([]byte("page-0"))
		case "2":
			_, _ = w.Write([]byte("page-1"))
		default:
			_, _ = w.Write([]byte("page-x"))
		}
	}))
	defer srv.Close()

	src := crawlTarget{source: storage.Source{Name: "test", BaseURL: srv.URL}, driver: fakeCatalogDriver{}}
	lister := catalogLister{fetch: NewFetcher()}

	t.Run("весь каталог до конца пагинации", func(t *testing.T) {
		urls, err := lister.list(context.Background(), src, 0)
		if err != nil {
			t.Fatalf("catalogLister.list: неожиданная ошибка: %v", err)
		}
		if len(urls) != 3 {
			t.Errorf("получено %d URL, ожидалось 3: %v", len(urls), urls)
		}
	})

	t.Run("лимит страниц", func(t *testing.T) {
		urls, err := lister.list(context.Background(), src, 1)
		if err != nil {
			t.Fatalf("catalogLister.list: неожиданная ошибка: %v", err)
		}
		if len(urls) != 2 {
			t.Errorf("получено %d URL, ожидалось 2: %v", len(urls), urls)
		}
	})
}

// ——— badPageDumper ———

func TestBadPageDumper(t *testing.T) {
	dir := t.TempDir()
	src := storage.Source{Name: "test", BaseURL: "http://example.com"}

	badPageDumper{dir: dir}.dump(src, "manga/some*weird:name", []byte("<html>битая</html>"), errors.New("разметка"))

	data, err := os.ReadFile(filepath.Join(dir, "example.com", "bad-pages", "manga_some_weird_name.html"))
	if err != nil {
		t.Fatalf("дамп не найден: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "parse error: разметка") || !strings.Contains(text, "<html>битая</html>") {
		t.Errorf("неожиданное содержимое дампа:\n%s", text)
	}
}

func TestBadPageDumperDisabled(t *testing.T) {
	// Пустой каталог — дампы отключены: не пишет и не паникует.
	badPageDumper{}.dump(storage.Source{BaseURL: "http://example.com"}, "manga/x", []byte("x"), nil)
}

// ——— уведомления ———

func TestFormatNewChaptersMessage(t *testing.T) {
	chapters := func(numbers ...string) []parsers.Chapter {
		result := make([]parsers.Chapter, 0, len(numbers))
		for _, n := range numbers {
			result = append(result, parsers.Chapter{Number: n})
		}
		return result
	}

	cases := []struct {
		name      string
		title     string
		chapters  []parsers.Chapter
		contains  []string
		notInText []string
	}{
		{
			name:     "одна глава",
			title:    "Манга",
			chapters: chapters("12"),
			contains: []string{"📚 <b>Манга</b> — новые главы!", "12", `<a href="http://site/manga/x">Читать</a>`},
		},
		{
			name:     "несколько глав",
			title:    "Манга",
			chapters: chapters("12", "13", "14"),
			contains: []string{"12, 13, 14"},
		},
		{
			name:      "больше пяти глав — свёрнуто в «и ещё»",
			title:     "Манга",
			chapters:  chapters("1", "2", "3", "4", "5", "6", "7"),
			contains:  []string{"1, 2, 3, 4, 5", "(и ещё 2)"},
			notInText: []string{"6", "7"},
		},
		{
			name:     "HTML в названии экранируется",
			title:    `<b>Ловушка</b> & "кавычки"`,
			chapters: chapters("1"),
			contains: []string{"&lt;b&gt;Ловушка&lt;/b&gt; &amp; &#34;кавычки&#34;"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text := formatNewChaptersMessage(tc.title, "http://site/manga/x", tc.chapters)

			for _, want := range tc.contains {
				if !strings.Contains(text, want) {
					t.Errorf("в тексте нет %q:\n%s", want, text)
				}
			}
			for _, absent := range tc.notInText {
				if strings.Contains(text, absent) {
					t.Errorf("в тексте не должно быть %q:\n%s", absent, text)
				}
			}
		})
	}
}

// ——— crawlTargets ———

// sourcesRepo — мок с управляемым списком источников.
type sourcesRepo struct {
	repo
	sources []storage.Source
}

func (r *sourcesRepo) GetSources(filter storage.SourceFilter) ([]storage.Source, error) {
	if filter.Enabled == nil || !*filter.Enabled {
		return nil, errors.New("краулер должен запрашивать только включённые источники")
	}
	return r.sources, nil
}

func TestCrawlTargets(t *testing.T) {
	t.Run("неизвестный движок пропускается, остальные обходятся", func(t *testing.T) {
		c := New(&sourcesRepo{sources: []storage.Source{
			{ID: 1, Name: "ок", BaseURL: "http://ok", Engine: "grouple"},
			{ID: 2, Name: "битый", BaseURL: "http://bad", Engine: "неизвестный"},
		}}, Config{})

		targets, err := c.crawlTargets()
		if err != nil {
			t.Fatalf("crawlTargets: неожиданная ошибка: %v", err)
		}
		if len(targets) != 1 || targets[0].source.Name != "ок" {
			t.Errorf("получено %v, ожидался только источник «ок»", targets)
		}
	})

	t.Run("ошибка БД пробрасывается вызывающему", func(t *testing.T) {
		c := New(&sourcesRepo{sources: nil}, Config{})
		// repo с nil-источником вернёт пустой список без ошибки:
		// пустой список — не ошибка чтения, её эмулируем отдельным моком.
		failing := failingSourcesRepo{}
		c2 := New(&failing, Config{})

		if _, err := c.crawlTargets(); err != nil {
			t.Errorf("пустой список — не ошибка, получено: %v", err)
		}
		if _, err := c2.crawlTargets(); err == nil {
			t.Error("ошибка чтения источников должна пробрасываться")
		}
	})
}

// failingSourcesRepo — мок с ошибкой чтения источников.
type failingSourcesRepo struct {
	repo
}

func (failingSourcesRepo) GetSources(filter storage.SourceFilter) ([]storage.Source, error) {
	return nil, errors.New("бд недоступна")
}
