package utils

import (
	"fmt"
	"net/url"
	"strings"
)

// ParsedMangaURL результат парсинга URL манги
type ParsedMangaURL struct {
	BaseURL   string // Базовый URL (например, https://a.zazaza.me)
	Host      string // Хост без протокола (например, a.zazaza.me)
	MangaPath string // Путь к манге (например, ugroza_v_moem_serdce__A5238)
}

// ParseMangaURL парсит URL манги и извлекает базовый URL и путь
// Пример: https://a.zazaza.me/ugroza_v_moem_serdce__A5238
// -> BaseURL: https://a.zazaza.me, Host: a.zazaza.me, MangaPath: ugroza_v_moem_serdce__A5238
func ParseMangaURL(rawURL string) (*ParsedMangaURL, error) {
	rawURL = strings.TrimSpace(rawURL)

	// Парсим URL
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("некорректный URL: %w", err)
	}

	// Проверяем схему
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("неподдерживаемая схема: %s", parsed.Scheme)
	}

	// Проверяем хост
	if parsed.Host == "" {
		return nil, fmt.Errorf("отсутствует хост в URL")
	}

	// Получаем путь (убираем начальный слеш)
	path := strings.TrimPrefix(parsed.Path, "/")

	// Убираем trailing slash если есть
	path = strings.TrimSuffix(path, "/")

	if path == "" {
		return nil, fmt.Errorf("отсутствует путь к манге в URL")
	}

	// Проверяем что путь не содержит вложенностей (только один сегмент)
	if strings.Contains(path, "/") {
		// Берём только первый сегмент
		parts := strings.Split(path, "/")
		path = parts[0]
	}

	return &ParsedMangaURL{
		BaseURL:   fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host),
		Host:      parsed.Host,
		MangaPath: path,
	}, nil
}
