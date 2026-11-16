package telegram

import (
	"errors"
	"fmt"
	"html"
	"log"
	"strings"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

const (
	findResultsLimit = 10
)

// handleFind обрабатывает команду /find <запрос>.
func (bot *Bot) handleFind(chatID int64, query string) {
	query = strings.TrimSpace(query)
	if query == "" {
		bot.sendMessageToChat(chatID, textFindUsage, true)
		return
	}

	results, err := bot.backend.SearchManga(query, findResultsLimit)
	if err != nil {
		log.Printf("Ошибка поиска по запросу %q: %v", query, err)
		bot.sendMessageToChat(chatID, textSearchError, true)
		return
	}

	if len(results) == 0 {
		bot.sendMessageToChat(chatID, fmt.Sprintf(textNothingFound, html.EscapeString(query)), true)
		return
	}

	bot.sendMessageWithKeyboard(
		chatID,
		formatFindResults(query, results),
		true,
		findResultsKeyboard(results, bot.miniAppURL),
	)
}

// getMangaForAction загружает мангу для команды подписки/отписки.
// ok=false — reply содержит готовый текст ответа: манга не найдена
// или произошла ошибка хранилища.
func (bot *Bot) getMangaForAction(mangaID int64) (manga storage.Manga, reply string, ok bool) {
	manga, err := bot.backend.GetManga(mangaID)
	if err == nil {
		return manga, "", true
	}
	if errors.Is(err, storage.ErrMangaNotFound) {
		return storage.Manga{}, fmt.Sprintf(textMangaNotFound, mangaID), false
	}
	log.Printf("Ошибка получения манги %d: %v", mangaID, err)
	return storage.Manga{}, textServerError, false
}

// subscribeManga подписывает чат на мангу и возвращает текст подтверждения.
func (bot *Bot) subscribeManga(chatID, mangaID int64) string {
	manga, reply, ok := bot.getMangaForAction(mangaID)
	if !ok {
		return reply
	}

	created, err := bot.backend.Subscribe(chatID, mangaID)
	if err != nil {
		if errors.Is(err, storage.ErrMangaNotFound) {
			// Мангу удалили между проверкой и подпиской (гонка с краулером).
			return fmt.Sprintf(textMangaNotFound, mangaID)
		}
		log.Printf("Ошибка подписки чата %d на мангу %d: %v", chatID, mangaID, err)
		return textServerError
	}

	if !created {
		return fmt.Sprintf(textAlreadyFollowed, mangaLink(manga))
	}

	return fmt.Sprintf(textFollowed, mangaLink(manga))
}

// unsubscribeManga отписывает чат от манги и возвращает текст подтверждения.
func (bot *Bot) unsubscribeManga(chatID, mangaID int64) string {
	manga, reply, ok := bot.getMangaForAction(mangaID)
	if !ok {
		return reply
	}

	deleted, err := bot.backend.Unsubscribe(chatID, mangaID)
	if err != nil {
		log.Printf("Ошибка отписки чата %d от манги %d: %v", chatID, mangaID, err)
		return textServerError
	}

	if !deleted {
		return fmt.Sprintf(textNotFollowed, mangaLink(manga))
	}

	return fmt.Sprintf(textUnfollowed, mangaLink(manga))
}

// handleList обрабатывает команду /list — подписки чата.
func (bot *Bot) handleList(chatID int64) {
	subscriptions, err := bot.backend.UserSubscriptions(chatID)
	if err != nil {
		log.Printf("Ошибка получения подписок чата %d: %v", chatID, err)
		bot.sendMessageToChat(chatID, textServerError, true)
		return
	}

	if len(subscriptions) == 0 {
		bot.sendMessageToChat(chatID, textEmptyList, true)
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 <b>Ваши подписки (%d):</b>\n", len(subscriptions)))

	for _, manga := range subscriptions {
		sb.WriteString(fmt.Sprintf(
			"\n• %s — ID: <code>%d</code>, глав: %d%s",
			mangaLink(manga), manga.ID, manga.ChaptersCount, mangaStatusSuffix(manga.PublicationStatus),
		))
	}

	bot.sendMessageWithKeyboard(chatID, sb.String(), true, subscriptionsKeyboard(subscriptions))
}
