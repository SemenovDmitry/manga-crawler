package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
)

// SendMessageToChat sends an HTML message to the given chat; a disabled preview strips the link snippet card.
// A disabled bot silently succeeds, mirroring the previous per-function behavior.
func SendMessageToChat(bot *TelegramBot, chatID string, text string, disablePreview bool) error {
	if !bot.Enabled {
		log.Println("Telegram бот отключен, сообщение не отправлено")
		return nil
	}

	message := Message{
		ChatID:                chatID,
		Text:                  text,
		ParseMode:             "HTML",
		DisableWebPagePreview: disablePreview,
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

	log.Printf("Telegram сообщение отправлено в чат %s", chatID)
	return nil
}

// SendMessage notifies the service chat configured in TELEGRAM_CHAT_ID, keeping the link preview enabled.
func sendMessage(bot *TelegramBot, text string) error {
	return SendMessageToChat(bot, bot.ChatID, text, false)
}

// SendMessageWithDisabledPreview notifies the service chat with the link preview disabled.
// func sendMessageWithDisabledPreview(bot *TelegramBot, text string) error {
// 	return SendMessageToChat(bot, bot.ChatID, text, true)
// }

// sendMessageToUser replies to the chat the message came from, with the link preview disabled.
func sendMessageToUser(bot *TelegramBot, chatID int64, text string) error {
	return SendMessageToChat(bot, fmt.Sprintf("%d", chatID), text, true)
}

// SendStartupNotification announces the bot startup in the service chat.
func SendStartupNotification(bot *TelegramBot) error {
	if !bot.Enabled {
		return nil
	}

	return sendMessage(bot, TextStartupNotification)
}
