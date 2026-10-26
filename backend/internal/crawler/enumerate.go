package crawler

import (
	"context"
	"encoding/xml"
	"fmt"
	"log"
	"strings"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers"
)

// maxCatalogPages — защитный предел страниц каталога за один обход
// (5000 × 50 = 250k карточек; реальный максимум сейчас — 575 страниц).
const maxCatalogPages = 5000

// urlLister — стратегия перечисления URL карточек манги в источнике:
// sitemap, пагинация каталога или их комбинация. Реализации не знают
// ни о проходах краулера, ни о БД.
type urlLister interface {
	// list возвращает URL карточек манги. limit > 0 ограничивает объём
	// сверху; 0 — перечислить всё (полный обход всегда идёт с 0).
	// При сбое возвращает собранное и ошибку: вызывающий использует
	// частичный список — недостающее доберут следующие проходы.
	list(ctx context.Context, src crawlTarget, limit int) ([]string, error)
}

// newSitemapLister и newCatalogLister возвращают реализации перечисления.
// Драйвер без parsers.CatalogLister (mgeko) в каталоге даёт пустой список —
// источник тогда работает только по sitemap.
func newSitemapLister(fetch *Fetcher, gate *pauseGate) urlLister {
	return sitemapLister{fetch: fetch, gate: gate}
}
func newCatalogLister(fetch *Fetcher, pace pacer, gate *pauseGate) urlLister {
	return catalogLister{fetch: fetch, pace: pace, gate: gate}
}

// ——— перечисление по sitemap ———

// sitemapIndex — корневой элемент индекса карт (карта карт).
type sitemapIndex struct {
	Sitemaps []sitemapRef `xml:"sitemap"`
}

// sitemapRef — ссылка на под-карту сайта.
type sitemapRef struct {
	Loc string `xml:"loc"`
}

// sitemapURLSet — корневой элемент карты сайта (<urlset>).
type sitemapURLSet struct {
	URLs []urlEntry `xml:"url"`
}

// urlEntry — один URL в карте сайта.
type urlEntry struct {
	Loc string `xml:"loc"`
}

// sitemapLister собирает URL всех страниц манги из sitemap источника
// (обходим индекс карт, если это он); отбор URL — по правилам драйвера.
type sitemapLister struct {
	fetch *Fetcher
	gate  *pauseGate // nil — уступки отключены (тесты)
}

func (l sitemapLister) list(ctx context.Context, src crawlTarget, _ int) ([]string, error) {
	baseURL := strings.TrimRight(src.source.BaseURL, "/")

	body, err := l.fetch.Page(ctx, baseURL+"/sitemap.xml")
	if err != nil {
		// Без /sitemap.xml пробуем карты из robots.txt (mangalib:
		// /sitemap.xml — SPA-заглушка, а карты перечислены в robots.txt).
		log.Printf("[%s] sitemap.xml недоступен (%v), пробую карты из robots.txt", src.source.Name, err)
		return l.urlsFromRobots(ctx, src, baseURL)
	}

	var locations []string

	var index sitemapIndex
	if err := xml.Unmarshal(body, &index); err == nil && len(index.Sitemaps) > 0 {
		for _, ref := range index.Sitemaps {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			// Точка уступки между под-картами: инкрементальный проход
			// не должен ждать скачивания всего индекса.
			if err := l.gate.yield(ctx); err != nil {
				return nil, err
			}
			locs, err := l.urlsFromSitemap(ctx, src, ref.Loc)
			if err != nil {
				continue // одна битая под-карта не отменяет обход
			}
			locations = append(locations, locs...)
		}
	} else {
		var set sitemapURLSet
		if uErr := xml.Unmarshal(body, &set); uErr != nil {
			// Ни индекс, ни карта: возможно, /sitemap.xml — заглушка SPA,
			// а карты перечислены в robots.txt (mangalib).
			log.Printf("[%s] sitemap.xml не распознан (%v), пробую карты из robots.txt", src.source.Name, uErr)
			return l.urlsFromRobots(ctx, src, baseURL)
		}
		for _, entry := range set.URLs {
			locations = append(locations, entry.Loc)
		}
	}

	var mangaURLs []string
	for _, loc := range locations {
		if src.driver.IsMangaURL(loc) {
			mangaURLs = append(mangaURLs, loc)
		}
	}

	return mangaURLs, nil
}

// urlsFromSitemap скачивает одну под-карту и возвращает её URL.
func (l sitemapLister) urlsFromSitemap(ctx context.Context, src crawlTarget, sitemapURL string) ([]string, error) {
	body, err := l.fetch.Page(ctx, sitemapURL)
	if err != nil {
		return nil, err
	}

	var set sitemapURLSet
	if err := xml.Unmarshal(body, &set); err != nil {
		return nil, fmt.Errorf("ошибка разбора %s: %w", sitemapURL, err)
	}

	locs := make([]string, 0, len(set.URLs))
	for _, entry := range set.URLs {
		locs = append(locs, entry.Loc)
	}

	return locs, nil
}

// urlsFromRobots достаёт адреса карт сайта из robots.txt (директивы
// Sitemap:) и собирает URL карточек из каждой карты. Одна битая карта
// не отменяет обход: недостающее доберут следующие проходы.
func (l sitemapLister) urlsFromRobots(ctx context.Context, src crawlTarget, baseURL string) ([]string, error) {
	body, err := l.fetch.Page(ctx, baseURL+"/robots.txt")
	if err != nil {
		return nil, fmt.Errorf("robots.txt недоступен: %w", err)
	}

	var sitemapURLs []string
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "Sitemap:"); ok {
			sitemapURLs = append(sitemapURLs, strings.TrimSpace(rest))
		}
	}
	if len(sitemapURLs) == 0 {
		return nil, fmt.Errorf("в robots.txt нет директив Sitemap")
	}
	log.Printf("[%s] В robots.txt указано карт сайта: %d", src.source.Name, len(sitemapURLs))

	var mangaURLs []string
	for _, sitemapURL := range sitemapURLs {
		if ctx.Err() != nil {
			return mangaURLs, ctx.Err()
		}
		if err := l.gate.yield(ctx); err != nil {
			return mangaURLs, err
		}
		locs, err := l.urlsFromSitemap(ctx, src, sitemapURL)
		if err != nil {
			continue // одна битая под-карта не отменяет обход
		}
		for _, loc := range locs {
			if src.driver.IsMangaURL(loc) {
				mangaURLs = append(mangaURLs, loc)
			}
		}
	}

	return mangaURLs, nil
}

// ——— перечисление по каталогу ———

// catalogLister собирает URL карточек манги из пагинации каталога.
type catalogLister struct {
	fetch *Fetcher
	pace  pacer
	gate  *pauseGate // nil — уступки отключены (тесты)
}

func (cl catalogLister) list(ctx context.Context, src crawlTarget, maxPages int) ([]string, error) {
	lister, ok := src.driver.(parsers.CatalogLister)
	if !ok {
		// Каталог используется только в полных обходах, здесь limit всегда 0.
		log.Printf("[%s] Драйвер не поддерживает обход каталога, перечисление только по sitemap", src.source.Name)
		return nil, nil
	}

	var urls []string
	seen := make(map[string]bool)
	pages := 0
	for offset := 0; ; offset += lister.CatalogPageSize() {
		if ctx.Err() != nil {
			return urls, ctx.Err()
		}
		if maxPages > 0 && pages >= maxPages {
			break
		}
		if pages >= maxCatalogPages {
			log.Printf("[%s] Достигнут защитный предел страниц каталога (%d), останавливаюсь",
				src.source.Name, maxCatalogPages)
			break
		}
		pages++

		pageURL := lister.CatalogPageURL(src.source.BaseURL, offset)
		body, err := cl.fetch.Page(ctx, pageURL)
		if err != nil {
			return urls, fmt.Errorf("страница каталога %s: %w", pageURL, err)
		}

		pageURLs := lister.MangaURLsFromCatalog(body, src.source.BaseURL)
		if len(pageURLs) == 0 {
			break // каталог кончился
		}

		fresh := 0
		for _, u := range pageURLs {
			if !seen[u] {
				seen[u] = true
				urls = append(urls, u)
				fresh++
			}
		}
		if fresh == 0 {
			// Сайт отдаёт уже показанные карточки — смещение за пределами
			// каталога или лимит выдачи; дальше ловить нечего.
			break
		}

		if pages%100 == 0 {
			log.Printf("[%s] Каталог: страниц %d, карточек %d", src.source.Name, pages, len(urls))
		}
		if err := cl.pace.wait(ctx); err != nil {
			return urls, err
		}
		// Точка уступки между страницами каталога.
		if err := cl.gate.yield(ctx); err != nil {
			return urls, err
		}
	}

	return urls, nil
}
