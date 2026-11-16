package parsers

import (
	"fmt"
	"net/url"
	"strings"
)

// URLPath — путь адреса без ограничивающих слэшей: url-часть манги
// ("asura_2024", "manga/slug", "ru/manga/123--slug"), она же ключ
// уникальности карточки (parse_source_id, url). Единая для всех движков:
// правило «что считать адресом манги» — это IsMangaURL драйвера поверх неё.
func URLPath(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("некорректный адрес %s: %w", rawURL, err)
	}

	return strings.Trim(u.Path, "/"), nil
}
