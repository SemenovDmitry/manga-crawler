package grouple

import (
	"os"
	"strings"
	"testing"
)

// Фрагмент повторяет разметку страницы каталога /list движка Grouple
// (см. probe-output/*/005-catalog-list.html): карточки — ссылки
// class="non-hover" внутри тайлов; служебные ссылки других классов
// и повторы не должны попадать в выборку.
const catalogFixture = `
<div class="tile col-sm-6 el_1">
  <a href="/van_pis__A5664" class="non-hover">
    <img class='lazy img-fluid hide-alt' src='' data-original='https://rm.one-way.work/uploads/pics/00/76/146_p.webp'>
  </a>
</div>
<div class="tile col-sm-6 el_2">
  <a href="/naruto__A35c96" class="non-hover">
    <img class='lazy img-fluid hide-alt' src=''>
  </a>
</div>
<div class="tile col-sm-6 el_3">
  <a href="/ganc" class="non-hover"><img class='lazy img-fluid hide-alt' src=''></a>
</div>
<a href="/logoff" class="btn btn-icon">Выход</a>
<a href="/collection" class="dropdown-item">Коллекции</a>
<div class="tile col-sm-6 el_4">
  <a href="/van_pis__A5664" class="non-hover"><img src=''></a>
</div>
`

func TestMangaURLsFromCatalog(t *testing.T) {
	parser := NewParser()
	urls := parser.MangaURLsFromCatalog([]byte(catalogFixture), "https://a.zazaza.me")

	want := []string{
		"https://a.zazaza.me/van_pis__A5664",
		"https://a.zazaza.me/naruto__A35c96",
		"https://a.zazaza.me/ganc",
	}

	if len(urls) != len(want) {
		t.Fatalf("получено %d URL (%v), ожидалось %d", len(urls), urls, len(want))
	}
	for i := range want {
		if urls[i] != want[i] {
			t.Errorf("URL %d: получено %q, ожидалось %q", i, urls[i], want[i])
		}
	}
}

func TestCatalogPageURL(t *testing.T) {
	parser := NewParser()

	if got, want := parser.CatalogPageURL("https://a.zazaza.me", 0),
		"https://a.zazaza.me/list?sortType=created&offset=0"; got != want {
		t.Errorf("первая страница: получено %q, ожидалось %q", got, want)
	}

	if got, want := parser.CatalogPageURL("https://a.zazaza.me/", 150),
		"https://a.zazaza.me/list?sortType=created&offset=150"; got != want {
		t.Errorf("страница со смещением: получено %q, ожидалось %q", got, want)
	}

	if parser.CatalogPageSize() != 50 {
		t.Errorf("размер страницы каталога: получено %d, ожидалось 50", parser.CatalogPageSize())
	}
}

// TestCatalogProbePage проверяет извлечение на реальной странице каталога
// из разведки (если probe-output отсутствует, тест пропускается).
func TestCatalogProbePage(t *testing.T) {
	files := []string{
		"../../../probe-output/a.zazaza.me/005-catalog-list.html",
		"../../../probe-output/1.seimanga.me/005-catalog-list.html",
	}

	for _, file := range files {
		body, err := os.ReadFile(file)
		if err != nil {
			if os.IsNotExist(err) {
				t.Skipf("нет файла разведки %s", file)
			}
			t.Fatalf("чтение %s: %v", file, err)
		}

		parser := NewParser()
		urls := parser.MangaURLsFromCatalog(body, "https://example.test")

		if len(urls) != parser.CatalogPageSize() {
			t.Errorf("%s: получено %d карточек, ожидалось %d", file, len(urls), parser.CatalogPageSize())
		}
		for _, u := range urls {
			part, err := parser.MangaURLPart(u)
			if err != nil || part == "" {
				t.Errorf("%s: некорректный URL карточки %q (ошибка %v)", file, u, err)
			}
			if !parser.IsMangaURL(u) {
				t.Errorf("%s: URL из каталога не прошёл фильтр sitemap: %s", file, u)
			}
			if !strings.HasPrefix(u, "https://example.test/") {
				t.Errorf("%s: URL не на базовом домене: %s", file, u)
			}
		}
	}
}
