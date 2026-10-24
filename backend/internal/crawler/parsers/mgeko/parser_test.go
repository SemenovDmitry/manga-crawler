package mgeko

import (
	"testing"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers"
)

// Тестовый фрагмент повторяет реальную разметку mgeko.cc (см.
// probe-output/www.mgeko.cc): h1.novel-title, h2.alternative-title,
// статус в <strong class="..."> перед <small>Status</small>, главы —
// li с data-volumeno и ссылкой /reader/...-chapter-<номер>-...
const mgekoCardFixture = `
<div class="novel-info">
  <h1 itemprop="name" class="novel-title text2row">
    Shadow Slave
  </h1>
  <h2 class="alternative-title text1row">
  섀도우 슬레이브 • Shadow Slave
  </h2>
</div>
<span>
<strong><i class="material-icons">bookmark</i> 8.0 K</strong>
<small>Bookmarked</small>
</span>
<span>
<strong class="ongoing">Ongoing</strong>
<small>Status</small>
</span>
<ul class="chapter-list" start="1">
<li data-chapterno="1" data-volumeno="0" data-orderno="604041" class="chapter-list-item">
<span class="chapter-no"><i class="far fa-eye-slash vieweye"></i></span>
<a href="/reader/en/j3pu-shadow-slave-chapter-13-eng-li/">
<div class="chapter-number">13-eng-li</div>
</a>
</li>
<li data-chapterno="1" data-volumeno="0" data-orderno="599783">
<a href="/reader/en/j3pu-shadow-slave-chapter-12.5-eng-li/" title="Chapter 12.5-eng-li">
<strong class="chapter-title">12.5-eng-li</strong>
</a>
</li>
<li data-chapterno="2" data-volumeno="2" data-orderno="579945">
<a href="/reader/en/j3pu-shadow-slave-chapter-1-eng-li/">
</a>
</li>
</ul>
`

func TestMgekoCard(t *testing.T) {
	driver := NewParser()
	card, err := driver.ParseMangaCard([]byte(mgekoCardFixture))
	if err != nil {
		t.Fatalf("ошибка парсинга: %v", err)
	}

	if card.Title != "Shadow Slave" {
		t.Errorf("название: получено %q, ожидалось %q", card.Title, "Shadow Slave")
	}

	wantAlt := "섀도우 슬레이브 / Shadow Slave"
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
	if normalized[0].Number != "1" || normalized[1].Number != "12.5" || normalized[2].Number != "13" {
		t.Errorf("порядок глав: получено %s, %s, %s; ожидалось 1, 12.5, 13",
			normalized[0].Number, normalized[1].Number, normalized[2].Number)
	}
	// data-volumeno="0" означает «без тома»
	if normalized[2].Volume != "" {
		t.Errorf("том главы 13: получено %q, ожидалось пусто", normalized[2].Volume)
	}
	if normalized[0].Volume != "2" {
		t.Errorf("том главы 1: получено %q, ожидалось %q", normalized[0].Volume, "2")
	}
}

func TestMgekoCardNotACard(t *testing.T) {
	driver := NewParser()
	if _, err := driver.ParseMangaCard([]byte("<html><body>Error 404</body></html>")); err == nil {
		t.Error("страница без карточки должна возвращать ошибку")
	}
}

func TestMgekoURLs(t *testing.T) {
	driver := NewParser()

	cases := []struct {
		url  string
		want bool
	}{
		{"https://www.mgeko.cc/manga/j3pu-shadow-slave/", true},
		{"https://www.mgeko.cc/manga/gintama/", true},
		{"https://www.mgeko.cc/manga/j3pu-shadow-slave/all-chapters/", false},
		{"https://www.mgeko.cc/manga/", false},
		{"https://www.mgeko.cc/browse-comics/", false},
		{"https://www.mgeko.cc/", false},
		{"https://www.mgeko.cc/manga/cover.webp", false},
		{"https://1.seimanga.me/asura_2024", false}, // чужой движок
	}

	for _, tc := range cases {
		if got := driver.IsMangaURL(tc.url); got != tc.want {
			t.Errorf("IsMangaURL(%s): получено %v, ожидалось %v", tc.url, got, tc.want)
		}
	}

	part, err := driver.MangaURLPart("https://www.mgeko.cc/manga/j3pu-shadow-slave/")
	if err != nil {
		t.Fatalf("MangaURLPart: %v", err)
	}
	if part != "manga/j3pu-shadow-slave" {
		t.Errorf("MangaURLPart: получено %q, ожидалось %q", part, "manga/j3pu-shadow-slave")
	}
}

// Статусы — классы блока Status движка mgeko (см. probe-output).
func TestMgekoMapEngineStatus(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"ongoing", "ongoing"},
		{"Ongoing", "ongoing"},
		{"completed", "completed"},
		{"hiatus", "hiatus"},
		{"dropped", "discontinued"},
		{"cancelled", "cancelled"},
		{"season-end", ""}, // класс без статуса завершения — не меняем
		{"", ""},
	}

	for _, tc := range cases {
		if got := mapEngineStatus(tc.in); got != tc.want {
			t.Errorf("mapEngineStatus(%q): получено %q, ожидалось %q", tc.in, got, tc.want)
		}
	}
}
