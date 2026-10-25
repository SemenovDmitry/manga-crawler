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
