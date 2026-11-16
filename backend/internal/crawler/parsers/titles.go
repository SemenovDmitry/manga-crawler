package parsers

import (
	"html"
	"regexp"
	"strings"
)

// Очистка текста названий: используется драйверами для полей
// MangaCard.Title, MangaCard.AltTitles и Chapter.Title (подписи глав).

var reHTMLTags = regexp.MustCompile(`<[^>]+>`)
var reSpaces = regexp.MustCompile(`\s+`)

// CleanTitle вычищает теги, HTML-сущности и лишние пробелы из текста
// названия (см. комментарий к файлу — какие поля это затрагивает).
func CleanTitle(raw string) string {
	text := reHTMLTags.ReplaceAllString(raw, " ")
	text = html.UnescapeString(text)
	return strings.TrimSpace(reSpaces.ReplaceAllString(text, " "))
}

// JoinUnique склеивает части альтернативного названия через " / ",
// отбрасывая пустые и повторяющиеся. Используется драйверами для
// MangaCard.AltTitles.
func JoinUnique(parts []string) string {
	var titles []string
	seen := make(map[string]bool)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		titles = append(titles, part)
	}

	return strings.Join(titles, " / ")
}
