package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/types"
)

// SendMessage отправка простого текстового сообщения
func SendMessage(bot *TelegramBot, text string) error {
	if !bot.Enabled {
		log.Println("Telegram бот отключен, сообщение не отправлено")
		return nil
	}

	message := Message{
		ChatID:    bot.ChatID,
		Text:      text,
		ParseMode: "HTML",
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга JSON: %v", err)
	}

	resp, err := bot.Client.Post(
		fmt.Sprintf("%s/sendMessage", bot.BotURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("ошибка отправки запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("ошибка парсинга ответа: %v", err)
	}

	if !apiResp.OK {
		return fmt.Errorf("ошибка Telegram API: %s", apiResp.Description)
	}

	log.Printf("Telegram сообщение отправлено: %s", text[:min(50, len(text))])
	return nil
}

// SendMangaUpdate отправка уведомления об обновлении манги
func SendMangaUpdate(bot *TelegramBot, sourceUrl string, manga *types.Manga, newChapters []types.Chapter) error {
	if !bot.Enabled || len(newChapters) == 0 {
		return nil
	}

	var message strings.Builder

	// Заголовок
	message.WriteString(fmt.Sprintf("%s\n\n", sourceUrl))

	if len(newChapters) > 1 {
		message.WriteString(fmt.Sprintf("<a href=\"%s\">%s</a>\n", manga.URL, escapeHTML(manga.Title)))
		message.WriteString(fmt.Sprintf("<b>Новые главы: %d</b>", len(newChapters)))
	} else {
		message.WriteString(fmt.Sprintf("<a href=\"%s\">%s</a>\n\n", newChapters[0].URL, newChapters[0].Title))
	}

	// Отправляем сообщение с ОТКЛЮЧЕННЫМ превью
	return sendMessageWithDisabledPreview(bot, message.String())
}

// sendMessageWithDisabledPreview отправка с отключенным превью
func sendMessageWithDisabledPreview(bot *TelegramBot, text string) error {
	if !bot.Enabled {
		return nil
	}

	message := Message{
		ChatID:                bot.ChatID,
		Text:                  text,
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга JSON: %v", err)
	}

	resp, err := bot.Client.Post(
		fmt.Sprintf("%s/sendMessage", bot.BotURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("ошибка отправки запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("ошибка парсинга ответа: %v", err)
	}

	if !apiResp.OK {
		return fmt.Errorf("ошибка Telegram API: %s", apiResp.Description)
	}

	log.Printf("Telegram сообщение отправлено (превью отключено)")
	return nil
}

// SendMangaUpdateToUser отправка уведомления конкретному пользователю
func SendMangaUpdateToUser(bot *TelegramBot, chatID int64, sourceUrl string, manga types.Manga, newChapters []types.Chapter) error {
	if !bot.Enabled || len(newChapters) == 0 {
		return nil
	}

	var messageText strings.Builder

	// Формируем полный URL манги
	mangaFullURL := fmt.Sprintf("%s/%s", sourceUrl, manga.URL)

	if len(newChapters) > 1 {
		messageText.WriteString(fmt.Sprintf("<a href=\"%s\">%s</a>\n", mangaFullURL, escapeHTML(manga.Title)))
		messageText.WriteString(fmt.Sprintf("<b>Новые главы: %d</b>\n\n", len(newChapters)))
		for _, ch := range newChapters {
			messageText.WriteString(fmt.Sprintf("• <a href=\"%s\">%s</a>\n", ch.URL, escapeHTML(ch.Title)))
		}
	} else {
		chapterURL := newChapters[0].URL
		messageText.WriteString(fmt.Sprintf("<a href=\"%s\">%s</a>\n", mangaFullURL, escapeHTML(manga.Title)))
		messageText.WriteString(fmt.Sprintf("Новая глава: <a href=\"%s\">%s</a>", chapterURL, escapeHTML(newChapters[0].Title)))
	}

	// Отправляем сообщение конкретному пользователю
	return sendMessageToUser(bot, chatID, messageText.String())
}

// sendMessageToUser отправка сообщения конкретному пользователю
func sendMessageToUser(bot *TelegramBot, chatID int64, text string) error {
	if !bot.Enabled {
		return nil
	}

	message := Message{
		ChatID:                fmt.Sprintf("%d", chatID),
		Text:                  text,
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга JSON: %v", err)
	}

	resp, err := bot.Client.Post(
		fmt.Sprintf("%s/sendMessage", bot.BotURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("ошибка отправки запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("ошибка парсинга ответа: %v", err)
	}

	if !apiResp.OK {
		return fmt.Errorf("ошибка Telegram API: %s", apiResp.Description)
	}

	log.Printf("Telegram сообщение отправлено пользователю %d", chatID)
	return nil
}

// SendNewMangaNotification уведомление о добавлении новой манги
func SendNewMangaNotification(bot *TelegramBot, manga *types.Manga) error {
	if !bot.Enabled {
		return nil
	}

	message := fmt.Sprintf(
		"✅ <b>Добавлена новая манга для отслеживания!</b>\n\n"+
			"<b>Название:</b> %s\n"+
			"<b>Ссылка:</b> <a href=\"%s\">Открыть</a>\n"+
			"<b>Количество глав:</b> %d\n\n"+
			"<i>Теперь буду отслеживать обновления этой манги!</i>",
		escapeHTML(manga.Title),
		manga.URL,
		len(manga.Chapters),
	)

	return SendMessage(bot, message)
}

// SendStartupNotification уведомление о запуске системы
func SendStartupNotification(bot *TelegramBot) error {
	if !bot.Enabled {
		return nil
	}

	message := "🚀 <b>Манга-трекер запущен!</b>\n\n" +
		"<i>Начинаю отслеживание обновлений...</i>"

	return SendMessage(bot, message)
}

// SendErrorNotification уведомление об ошибке
func SendErrorNotification(bot *TelegramBot, mangaName string) error {
	if !bot.Enabled {
		return nil
	}

	message := fmt.Sprintf("🚨 <b>Ошибка парсинга RSS ленты - %s</b>\n", mangaName)

	return SendMessage(bot, message)
}

// escapeHTML экранирование HTML символов
func escapeHTML(text string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(text)
}
