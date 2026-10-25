package grouple

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers"
)

// Тестовый фрагмент повторяет реальную разметку движка (см. probe-output):
// «Ганц» — data-translation-status="FINISHED", «Ван Пис» — "PROGRESS".
const cardFixture = `
<div class="cr-hero-names">
  <h1 class="cr-hero-names__main line-clamp-2">Тестовая манга</h1>
  <h3 class="cr-hero-names__alt text-truncate" data-bs-toggle="modal" data-bs-target="#alt-names-dialog">
    <span data-tippy-content="Подсказка">
      <span>Test Manga</span>
      <span class="cr-hero-names__alt-separator">/</span>
    </span>
    <span>Тест / Второе название / Тест</span>
  </h3>
</div>
<div class="cr-info-details-item__content">
  <span class="cr-info-details-item__status" data-production-status="FINISHED">Завершён</span>
</div>
<div class="cr-info-details-item__content">
  <span class="cr-info-details-item__status" data-translation-status="FINISHED">Завершён</span>
</div>
<a href="/tmanga/vol2/3" title="Team" class="chapter-link btn btn-primary btn-block cr-actions__btn-content-view read-first-chapter text-truncate manga-mtr">кнопка чтения</a>
<tr class="item-row" data-id="1" data-vol="2" data-num="30">
  <td class=" item-title" data-vol="2" data-num="30">
    <a href="/tmanga/vol2/3" title="Team" class="chapter-link cp-l manga-mtr">
      2 - 3 Последняя глава
    </a>
  </td>
</tr>
<tr class="item-row" data-id="2" data-vol="2" data-num="20">
  <td class=" item-title" data-vol="2" data-num="20">
    <a href="/tmanga/vol2/2" title="Team" class="chapter-link cp-l manga-mtr">
      2 - 2 <em>Глава с тегом</em>
    </a>
  </td>
</tr>
`

func TestParseMangaCardFixture(t *testing.T) {
	parser := NewParser()
	card, err := parser.ParseMangaCard([]byte(cardFixture))
	if err != nil {
		t.Fatalf("ошибка парсинга: %v", err)
	}

	if card.Title != "Тестовая манга" {
		t.Errorf("название: получено %q, ожидалось %q", card.Title, "Тестовая манга")
	}

	wantAlt := "Test Manga / Тест / Второе название"
	if card.AltTitles != wantAlt {
		t.Errorf("альт. названия: получено %q, ожидалось %q", card.AltTitles, wantAlt)
	}

	if card.PublicationStatus != "completed" {
		t.Errorf("статус публикации: получено %q, ожидалось %q", card.PublicationStatus, "completed")
	}

	normalized := parsers.NormalizeChapters(card.Chapters)
	if len(normalized) != 2 {
		t.Fatalf("глав: получено %d, ожидалось 2 (кнопка чтения не должна попадать)", len(normalized))
	}
	if normalized[0].Number != "2" || normalized[1].Number != "3" {
		t.Errorf("порядок глав: получено %s, %s; ожидалось 2, 3", normalized[0].Number, normalized[1].Number)
	}
	if normalized[1].Volume != "2" {
		t.Errorf("том главы: получено %q, ожидалось %q", normalized[1].Volume, "2")
	}
	if normalized[1].Title != "Последняя глава" {
		t.Errorf("подпись главы: получено %q, ожидалось %q", normalized[1].Title, "Последняя глава")
	}
	if normalized[0].Title != "Глава с тегом" {
		t.Errorf("подпись с тегами: получено %q, ожидалось %q", normalized[0].Title, "Глава с тегом")
	}

	newChapters := parsers.FindNewChapters(normalized, 1)
	if len(newChapters) != 1 || newChapters[0].Number != "3" {
		t.Errorf("новые главы: получено %+v, ожидалось [3]", newChapters)
	}
}

func TestParseMangaCardNotMangaPage(t *testing.T) {
	parser := NewParser()
	if _, err := parser.ParseMangaCard([]byte("<html><body>Служебная страница</body></html>")); err == nil {
		t.Fatal("ожидалась ошибка для страницы без карточки манги")
	}
}

// Удалённые манги движок отдаёт с HTTP 200 и пометкой в <title>
// (реальный случай: https://1.seimanga.me/suck_it, см. probe -url).
func TestParseMangaCardRemovedPage(t *testing.T) {
	parser := NewParser()

	removedPage := `<!doctype html>
<html lang='ru'>
<head>
  <title>Манга Пососи противоречит правилам сайта - SeiManga</title>
</head>
<body></body>
</html>`

	card, err := parser.ParseMangaCard([]byte(removedPage))
	if !errors.Is(err, parsers.ErrMangaRemoved) {
		t.Fatalf("ожидалась parsers.ErrMangaRemoved, получено: %v", err)
	}
	if card != nil {
		t.Fatalf("карточка должна быть nil, получено: %+v", card)
	}

	// Каждая из 6300 страниц содержит <title> — обычная карточка не должна
	// распознаваться как удалённая
	cardOK, err := parser.ParseMangaCard([]byte(cardFixture))
	if err != nil {
		t.Fatalf("обычная карточка распознана как удалённая: %v", err)
	}
	if cardOK.Title != "Тестовая манга" {
		t.Errorf("название обычной карточки: получено %q", cardOK.Title)
	}
}

// Проверка на реальных страницах из разведки (тест пропускается, если файлов нет).
func TestParseMangaCardProbeFiles(t *testing.T) {
	cases := []struct {
		file        string
		wantTitle   string
		wantStatus  string
		minChapters int
	}{
		{"../../../probe-output/1.seimanga.me/004-manga-3-ganc.html", "Ганц", "completed", 300},
		{"../../../probe-output/a.zazaza.me/004-manga-3-van_pis__A5664.html", "Ван Пис", "ongoing", 1100},
	}

	parser := NewParser()
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			body, err := os.ReadFile(tc.file)
			if err != nil {
				t.Skipf("файл разведки недоступен: %v", err)
			}

			card, err := parser.ParseMangaCard(body)
			if err != nil {
				t.Fatalf("ошибка парсинга: %v", err)
			}

			if card.Title != tc.wantTitle {
				t.Errorf("название: получено %q, ожидалось %q", card.Title, tc.wantTitle)
			}
			if card.PublicationStatus != tc.wantStatus {
				t.Errorf("статус публикации: получено %q, ожидалось %q", card.PublicationStatus, tc.wantStatus)
			}
			if len(card.Chapters) < tc.minChapters {
				t.Errorf("глав: получено %d, ожидалось не менее %d", len(card.Chapters), tc.minChapters)
			}
			if card.AltTitles == "" {
				t.Error("альтернативные названия пустые")
			} else if strings.Contains(card.AltTitles, "  ") {
				t.Errorf("двойные пробелы в альт. названиях: %q", card.AltTitles)
			}
		})
	}
}
