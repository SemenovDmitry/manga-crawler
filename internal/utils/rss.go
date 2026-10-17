package utils

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/types"
)

func FindRSSLink(baseUrl, mangaName string) (string, error) {
	// Формируем URL страницы манги
	mangaUrl := fmt.Sprintf("%s/%s", baseUrl, mangaName)

	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:          10,
			IdleConnTimeout:       60 * time.Second,
			DisableCompression:    false,
			DisableKeepAlives:     false,
			MaxIdleConnsPerHost:   10,
			TLSHandshakeTimeout:   15 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			ExpectContinueTimeout: 5 * time.Second,
		},
	}

	req, err := http.NewRequest("GET", mangaUrl, nil)
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %v", err)
	}

	req.Header = GetClientHeaders(baseUrl)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка запроса страницы: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("статус ошибки: %d %s", resp.StatusCode, resp.Status)
	}

	// Читаем и декомпрессируем тело
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	decompressedBody, err := DecompressGzipBody(body)
	if err != nil {
		log.Printf("Ошибка декомпрессии: %v", err)
		decompressedBody = body
	}

	// Ищем RSS ссылку в HTML
	htmlContent := string(decompressedBody)

	// Паттерн для поиска RSS ссылок
	patterns := []*regexp.Regexp{
		// Ищем <link> с атрибутом title="RSS"
		regexp.MustCompile(`<link[^>]*title=["']?RSS["']?[^>]*href=["']?([^"'\s>]+)["']?`),
		// Ищем <link> с type="application/rss+xml"
		regexp.MustCompile(`<link[^>]*type=["']?application/rss\+xml["']?[^>]*href=["']?([^"'\s>]+)["']?`),
		// Ищем <link> с rel="alternate" и type содержит rss
		regexp.MustCompile(`<link[^>]*rel=["']?alternate["']?[^>]*type=["']?application/rss\+xml["']?[^>]*href=["']?([^"'\s>]+)["']?`),
		// Ищем <a> с RSS в тексте или href
		regexp.MustCompile(`<a[^>]*href=["']?([^"'\s>]*rss[^"'\s>]*)["']?[^>]*>.*?RSS.*?</a>`),
	}

	var rssLink string

	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(htmlContent)
		if len(matches) > 1 {
			rssLink = matches[1]

			// Если ссылка относительная, делаем её абсолютной
			if !strings.HasPrefix(rssLink, "http") {
				parsedBaseUrl, err := url.Parse(baseUrl)
				if err != nil {
					return rssLink, nil
				}

				parsedRssLink, err := url.Parse(rssLink)
				if err != nil {
					return rssLink, nil
				}

				rssLink = parsedBaseUrl.ResolveReference(parsedRssLink).String()
			}

			log.Printf("Найдена RSS ссылка для %s: %s", mangaName, rssLink)
			return rssLink, nil
		}
	}

	if rssLink != "" {
		return rssLink, nil
	}

	// Если не нашли через регулярки, пытаемся найти в head
	headStart := strings.Index(htmlContent, "<head")
	if headStart != -1 {
		headEnd := strings.Index(htmlContent, "</head>")
		if headEnd != -1 {
			headSection := htmlContent[headStart:headEnd]

			// Ищем ссылку на RSS в head
			rssPattern := regexp.MustCompile(`href=["']?([^"'\s>]*rss[^"'\s>]*)["']?`)
			matches := rssPattern.FindAllStringSubmatch(headSection, -1)

			for _, match := range matches {
				if len(match) > 1 && strings.Contains(strings.ToLower(match[1]), "rss") {
					rssLink := match[1]

					// Если ссылка относительная, делаем её абсолютной
					if !strings.HasPrefix(rssLink, "http") {
						parsedBaseUrl, err := url.Parse(baseUrl)
						if err != nil {
							return rssLink, nil
						}

						parsedRssLink, err := url.Parse(rssLink)
						if err != nil {
							return rssLink, nil
						}

						rssLink = parsedBaseUrl.ResolveReference(parsedRssLink).String()
					}

					log.Printf("Найдена RSS ссылка в head для %s: %s", mangaName, rssLink)
					return rssLink, nil
				}
			}
		}
	}

	if rssLink == "" {
		return "", fmt.Errorf("RSS ссылка не найдена для %s", mangaName)
	}

	return rssLink, nil
}

func GetRSSFeed(baseUrl string) (types.Channel, error) {
	var channel types.Channel
	url := baseUrl

	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:          10,
			IdleConnTimeout:       60 * time.Second,
			DisableCompression:    false,
			DisableKeepAlives:     false,
			MaxIdleConnsPerHost:   10,
			TLSHandshakeTimeout:   15 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			ExpectContinueTimeout: 5 * time.Second,
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return channel, fmt.Errorf("ошибка создания запроса: %v", err)
	}

	// Важные заголовки
	req.Header = GetRSSHeaders(baseUrl)

	resp, err := client.Do(req)
	if err != nil {
		return channel, fmt.Errorf("ошибка запроса RSS: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return channel, fmt.Errorf("статус ошибки: %d %s", resp.StatusCode, resp.Status)
	}

	// Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return channel, fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	// Декомпрессируем если нужно
	decompressedBody, err := DecompressGzipBody(body)
	if err != nil {
		log.Printf("Ошибка декомпрессии: %v", err)
		decompressedBody = body
	}

	// Убираем BOM (Byte Order Mark) если есть
	decompressedBody = bytes.TrimPrefix(decompressedBody, []byte{0xEF, 0xBB, 0xBF})
	decompressedBody = bytes.TrimPrefix(decompressedBody, []byte{0xFE, 0xFF})
	decompressedBody = bytes.TrimPrefix(decompressedBody, []byte{0xFF, 0xFE})

	// Парсим XML
	var rss types.RSS
	if err := xml.Unmarshal(decompressedBody, &rss); err != nil {
		// Логируем первые 300 байт для отладки
		sample := string(decompressedBody)
		if len(sample) > 300 {
			sample = sample[:300]
		}
		log.Printf("Ошибка парсинга XML: %v", err)
		log.Printf("Начало данных: %s", sample)

		// Пробуем исправить XML - убираем невалидные символы
		if err := xml.Unmarshal(decompressedBody, &rss); err != nil {
			return channel, fmt.Errorf("ошибка парсинга XML после очистки: %v", err)
		}
	}

	return rss.Channel, nil
}

func TransformRSSFeed(feed types.Channel) (*types.Manga, error) {
	manga := &types.Manga{
		Title:    feed.Title,
		URL:      feed.Link,
		Chapters: make([]types.Chapter, 10),
	}

	if len(feed.Items) == 0 {
		return manga, nil
	}

	filteredItems := feed.Items[:min(len(feed.Items), 10)]

	for i, item := range filteredItems {
		manga.Chapters[i] = types.Chapter{
			Title: item.Title,
			URL:   item.Link,
		}
	}

	return manga, nil
}

// GetRSSHeaders возвращает заголовки для запросов RSS
func GetRSSHeaders(baseUrl string) http.Header {
	return http.Header{
		"User-Agent":      {GetRandomUserAgent()},
		"Accept":          {"application/xml,text/xml,application/rss+xml"},
		"Accept-Language": {"ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7"},
		"Accept-Encoding": {"gzip, deflate"},
		"Referer":         {baseUrl + "/"},
		"Connection":      {"keep-alive"},
	}
}
