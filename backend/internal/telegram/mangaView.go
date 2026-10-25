package telegram

import (
	"fmt"
	"html"
	"net/url"
	"strings"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

// Префиксы callback_data для inline-кнопок — без двоеточия.
// Полные данные кнопки: "<префикс>:<ID манги>"; парсер (handleCallbackQuery)
// сравнивает префикс после strings.Cut.
const (
	callbackFollow   = "follow"
	callbackUnfollow = "unfollow"
)

// mangaFullURL собирает полный адрес страницы манги.
func mangaFullURL(manga storage.Manga) string {
	if strings.HasPrefix(manga.URL, "http://") || strings.HasPrefix(manga.URL, "https://") {
		return manga.URL
	}

	return strings.TrimRight(manga.BaseURL, "/") + "/" + strings.TrimLeft(manga.URL, "/")
}

// mangaLink — HTML-ссылка на мангу для сообщений.
func mangaLink(manga storage.Manga) string {
	return fmt.Sprintf(`<a href="%s">%s</a>`, mangaFullURL(manga), html.EscapeString(manga.Title))
}

// mangaStatusSuffix — подпись статуса публикации для не-ongoing манг.
func mangaStatusSuffix(status storage.PublicationStatus) string {
	switch status {
	case storage.StatusCompleted:
		return " · 🏁 завершён"
	case storage.StatusHiatus:
		return " · ⏸ приостановлен"
	case storage.StatusCancelled:
		return " · 🚫 отменён"
	case storage.StatusDiscontinued:
		return " · ⛔ прекращён"
	default:
		return ""
	}
}

// mangaSourceHost — короткое имя сайта источника для выдачи поиска
// ("a.zazaza.me"): хост из base_url, без схемы и www-префикса.
func mangaSourceHost(manga storage.Manga) string {
	u, err := url.Parse(manga.BaseURL)
	if err != nil || u.Host == "" {
		return manga.SourceName
	}

	return strings.TrimPrefix(u.Host, "www.")
}

// truncatedButtonText обрезает текст кнопки до безопасной длины
// (лимит Telegram на текст кнопки — 128 символов, обрезаем с запасом).
func truncatedButtonText(title string) string {
	title = strings.Join(strings.Fields(title), " ")

	const maxRunes = 48
	runes := []rune(title)
	if len(runes) <= maxRunes {
		return title
	}

	return string(runes[:maxRunes]) + "…"
}

func formatFindResults(query string, results []storage.Manga) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔍 <b>Результаты по «%s»:</b>\n", html.EscapeString(query)))

	for i, manga := range results {
		sb.WriteString(fmt.Sprintf(
			"\n%d. %s — ID: <code>%d</code>, %s, глав: %d%s",
			i+1, mangaLink(manga), manga.ID, mangaSourceHost(manga), manga.ChaptersCount, mangaStatusSuffix(manga.PublicationStatus),
		))
	}

	sb.WriteString("\n\n")
	sb.WriteString(textFindButtonsHint)

	return sb.String()
}

// findResultsKeyboard строит inline-клавиатуру: кнопка на каждый результат.
func findResultsKeyboard(results []storage.Manga) *InlineKeyboardMarkup {
	rows := make([][]InlineKeyboardButton, 0, len(results))

	for i, manga := range results {
		rows = append(rows, []InlineKeyboardButton{{
			Text:         fmt.Sprintf("%d. %s", i+1, truncatedButtonText(manga.Title)),
			CallbackData: fmt.Sprintf("%s:%d", callbackFollow, manga.ID),
		}})
	}

	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

// subscriptionsKeyboard строит клавиатуру отписки для списка подписок.
func subscriptionsKeyboard(subscriptions []storage.Manga) *InlineKeyboardMarkup {
	rows := make([][]InlineKeyboardButton, 0, len(subscriptions))

	for _, manga := range subscriptions {
		rows = append(rows, []InlineKeyboardButton{{
			Text:         fmt.Sprintf("❌ %s", truncatedButtonText(manga.Title)),
			CallbackData: fmt.Sprintf("%s:%d", callbackUnfollow, manga.ID),
		}})
	}

	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}
