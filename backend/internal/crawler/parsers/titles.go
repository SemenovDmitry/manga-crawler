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
