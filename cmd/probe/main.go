// cmd/probe — утилита разведки структуры сайтов для краулера.
//
// Сайты блокируют датацентровые IP, поэтому запускать нужно на машине
// пользователя: go run ./cmd/probe
//
// Для каждого источника сохраняет в probe-output/<host>/:
//   - robots.txt
//   - sitemap.xml (и до 2 под-карт, если это sitemap index)
//   - примеры URL манги (первые 50 из карты)
//   - 3 примера страниц манги
//   - страницу каталога (/list)
//   - страницу поиска (/search?q=...)
//
// Собранные файлы используются для реализации парсера.
package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	defaultOutDir    = "probe-output"
	defaultQuery     = "one piece"

	maxResponseBody = 25 << 20 // 25 МБ
	maxSubSitemaps  = 2
	maxMangaSamples = 3
	urlSampleLimit  = 50
)

var defaultSources = []string{"https://1.seimanga.me", "https://a.zazaza.me"}

// sitemapIndex — корневой элемент sitemap index (карта карт).
type sitemapIndex struct {
	Sitemaps []sitemapRef `xml:"sitemap"`
}

// sitemapRef — ссылка на под-карту в sitemap index.
type sitemapRef struct {
	Loc string `xml:"loc"`
}

// urlSet — корневой элемент обычной карты сайта.
type urlSet struct {
	URLs []urlEntry `xml:"url"`
}

// urlEntry — один URL в карте сайта.
type urlEntry struct {
	Loc     string `xml:"loc"`
	Lastmod string `xml:"lastmod"`
}

type probeClient struct {
	http *http.Client
}

func newProbeClient() *probeClient {
	return &probeClient{http: &http.Client{Timeout: 45 * time.Second}}
}

func (c *probeClient) fetch(rawURL string) (int, []byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ru,en;q=0.8")

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("ошибка запроса: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	return resp.StatusCode, body, nil
}

var slugRe = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// slugFromURL превращает URL манги в безопасное имя файла.
func slugFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "page"
	}

	s := strings.Trim(u.Path, "/")
	if s == "" {
		s = u.Host
	}
	s = strings.ReplaceAll(s, "/", "_")
	if len(s) > 80 {
		s = s[:80]
	}

	return slugRe.ReplaceAllString(s, "_")
}

func saveFile(dir, name string, data []byte) string {
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		log.Printf("  ошибка записи %s: %v", path, err)
		return ""
	}
	return path
}

func isMangaURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}

	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(segments) != 1 || segments[0] == "" {
		return false // страница манги — один сегмент пути без вложенности
	}

	segment := segments[0]
	if strings.Contains(segment, ".") { // статика/файлы — не манга
		return false
	}

	excluded := map[string]bool{
		"list": true, "search": true, "rss": true, "api": true,
		"login": true, "random": true, "types": true, "genres": true,
	}

	return !excluded[segment]
}

// probeSource собирает примеры страниц одного сайта и складывает их в outDir/<host>.
func probeSource(c *probeClient, base, outDir, query string) error {
	base = strings.TrimRight(base, "/")

	u, err := url.Parse(base)
	if err != nil {
		return fmt.Errorf("некорректный адрес источника: %w", err)
	}

	dir := filepath.Join(outDir, u.Host)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("ошибка создания каталога %s: %w", dir, err)
	}

	log.Printf("[%s] Разведка начата, каталог: %s", u.Host, dir)
	var summary []string
	add := func(line string) {
		summary = append(summary, line)
		log.Printf("  [%s] %s", u.Host, line)
	}

	// 1. robots.txt
	status, body, err := c.fetch(base + "/robots.txt")
	if err != nil {
		add(fmt.Sprintf("robots.txt: ошибка: %v", err))
	} else {
		saveFile(dir, "001-robots.txt", body)
		add(fmt.Sprintf("robots.txt: HTTP %d, %d байт", status, len(body)))
	}

	// 2. sitemap.xml
	sampleURLs := probeSitemap(c, base, dir, add)

	// 3. Примеры страниц манги
	samples := 0
	for _, raw := range sampleURLs {
		if samples >= maxMangaSamples {
			break
		}
		status, body, err := c.fetch(raw)
		if err != nil {
			add(fmt.Sprintf("страница манги %s: ошибка: %v", raw, err))
			continue
		}
		name := fmt.Sprintf("004-manga-%d-%s.html", samples+1, slugFromURL(raw))
		saveFile(dir, name, body)
		add(fmt.Sprintf("страница манги %s: HTTP %d -> %s (%d байт)", raw, status, name, len(body)))
		samples++
	}

	// 4. Каталог
	status, body, err = c.fetch(base + "/list")
	if err != nil {
		add(fmt.Sprintf("каталог /list: ошибка: %v", err))
	} else {
		name := "005-catalog-list.html"
		saveFile(dir, name, body)
		add(fmt.Sprintf("каталог /list: HTTP %d -> %s (%d байт)", status, name, len(body)))
	}

	// 5. Поиск
	searchURL := base + "/search?q=" + url.QueryEscape(query)
	status, body, err = c.fetch(searchURL)
	if err != nil {
		add(fmt.Sprintf("поиск: ошибка: %v", err))
	} else {
		name := "006-search.html"
		saveFile(dir, name, body)
		add(fmt.Sprintf("поиск %q: HTTP %d -> %s (%d байт)", query, status, name, len(body)))
	}

	// Итоговая сводка
	summaryText := fmt.Sprintf("Источник: %s\nРазведка: %s\n\n%s\n",
		base, time.Now().Format(time.RFC3339), strings.Join(summary, "\n"))
	saveFile(dir, "000-summary.txt", []byte(summaryText))

	log.Printf("[%s] Разведка завершена", u.Host)
	return nil
}

// probeSitemap скачивает sitemap.xml; если это индекс — дополнительно до 2 под-карт.
// Возвращает примеры URL манги из карт.
func probeSitemap(c *probeClient, base, dir string, add func(string)) []string {
	status, body, err := c.fetch(base + "/sitemap.xml")
	if err != nil {
		add(fmt.Sprintf("sitemap.xml: ошибка: %v", err))
		return nil
	}

	saveFile(dir, "002-sitemap.xml", body)
	add(fmt.Sprintf("sitemap.xml: HTTP %d, %d байт", status, len(body)))

	// Сначала пробуем как sitemap index
	var index sitemapIndex
	if err := xml.Unmarshal(body, &index); err == nil && len(index.Sitemaps) > 0 {
		add(fmt.Sprintf("sitemap.xml: это индекс, под-карт: %d", len(index.Sitemaps)))

		var samples []string
		for i, ref := range index.Sitemaps {
			if i >= maxSubSitemaps {
				add(fmt.Sprintf("под-карт всего %d, скачаны первые %d", len(index.Sitemaps), maxSubSitemaps))
				break
			}
			samples = append(samples, probeSubSitemap(c, ref.Loc, dir, i+1, add)...)
		}
		return samples
	}

	// Возможно, карта плоская
	var set urlSet
	if err := xml.Unmarshal(body, &set); err != nil {
		add(fmt.Sprintf("sitemap.xml: не удалось разобрать XML: %v", err))
		return nil
	}

	add(fmt.Sprintf("sitemap.xml: плоская карта, URL: %d", len(set.URLs)))
	return extractSamples(set.URLs, dir, add)
}

// probeSubSitemap скачивает под-карту и возвращает примеры URL манги.
func probeSubSitemap(c *probeClient, rawURL, dir string, num int, add func(string)) []string {
	status, body, err := c.fetch(rawURL)
	if err != nil {
		add(fmt.Sprintf("под-карта %s: ошибка: %v", rawURL, err))
		return nil
	}

	name := fmt.Sprintf("003-sitemap-child-%d.xml", num)
	saveFile(dir, name, body)

	var set urlSet
	if err := xml.Unmarshal(body, &set); err != nil {
		add(fmt.Sprintf("под-карта %s: HTTP %d, %d байт, ошибка XML: %v", rawURL, status, len(body), err))
		return nil
	}

	add(fmt.Sprintf("под-карта %s: HTTP %d -> %s, URL: %d", rawURL, status, name, len(set.URLs)))
	return extractSamples(set.URLs, dir, add)
}

// extractSamples сохраняет первые urlSampleLimit URL в текстовый файл
// и возвращает первые валидные URL манги.
func extractSamples(entries []urlEntry, dir string, add func(string)) []string {
	var lines, samples []string
	for _, entry := range entries {
		if len(lines) < urlSampleLimit {
			line := entry.Loc
			if entry.Lastmod != "" {
				line += " | lastmod: " + entry.Lastmod
			}
			lines = append(lines, line)
		}
		if isMangaURL(entry.Loc) {
			samples = append(samples, entry.Loc)
		}
	}

	saveFile(dir, "007-sitemap-urls-sample.txt", []byte(strings.Join(lines, "\n")))
	add(fmt.Sprintf("примеры URL: сохранено %d строк в 007-sitemap-urls-sample.txt", len(lines)))

	return samples
}

func main() {
	outDir := flag.String("out", defaultOutDir, "каталог для сохранения результатов")
	query := flag.String("query", defaultQuery, "поисковый запрос для страницы поиска")
	debugURL := flag.String("url", "", "скачать одну страницу и сохранить в out/debug-*.html (режим отладки парсера)")
	flag.Parse()

	log.SetFlags(log.Ltime)
	c := newProbeClient()

	// Режим отладки: только указанная страница
	if *debugURL != "" {
		if err := probeSingleURL(c, *debugURL, *outDir); err != nil {
			log.Fatalf("Ошибка загрузки %s: %v", *debugURL, err)
		}
		return
	}

	sources := flag.Args()
	if len(sources) == 0 {
		sources = defaultSources
	}

	log.Printf("Разведка сайтов: %s", strings.Join(sources, ", "))

	for _, source := range sources {
		if err := probeSource(c, source, *outDir, *query); err != nil {
			log.Printf("Ошибка разведки %s: %v", source, err)
		}
	}

	log.Printf("Готово. Результаты в каталоге %s, пришлите его содержимое для реализации парсера", *outDir)
}

// probeSingleURL скачивает одну страницу в outDir для разбора проблемных случаев.
func probeSingleURL(c *probeClient, rawURL, outDir string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("некорректный адрес: %w", err)
	}

	dir := filepath.Join(outDir, u.Host)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("ошибка создания каталога %s: %w", dir, err)
	}

	status, body, err := c.fetch(rawURL)
	if err != nil {
		return err
	}

	name := fmt.Sprintf("debug-%s.html", slugFromURL(rawURL))
	saveFile(dir, name, body)
	log.Printf("[%s] HTTP %d, %d байт -> %s", u.Host, status, len(body), filepath.Join(dir, name))

	return nil
}
