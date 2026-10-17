package telegram

import (
	"encoding/json"
	"fmt"
	"log"
)

// SendMessageToChat sends an HTML message to the given chat; a disabled preview strips the link snippet card.
// A disabled bot silently succeeds, mirroring the previous per-function behavior.
func (bot *TelegramBot) SendMessageToChat(chatID string, text string, disablePreview bool) error {
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

	resp, err := bot.postJSON("sendMessage", jsonData)
	if err != nil {
		return fmt.Errorf("ошибка отправки запроса: %v", err)
	}

	var apiResp APIResponse
	if err := decodeAPIResponse(resp, &apiResp); err != nil {
		return err
	}

	if !apiResp.OK {
		return fmt.Errorf("ошибка Telegram API: %s", apiResp.Description)
	}

	log.Printf("Telegram сообщение отправлено в чат %s", chatID)
	return nil
}

// sendMessage notifies the service chat configured in TELEGRAM_CHAT_ID, keeping the link preview enabled.
func (bot *TelegramBot) sendMessage(text string) error {
	return bot.SendMessageToChat(bot.ChatID, text, false)
}

// SendMessageWithDisabledPreview notifies the service chat with the link preview disabled.
// func (bot *TelegramBot) sendMessageWithDisabledPreview(text string) error {
// 	return bot.SendMessageToChat(bot.ChatID, text, true)
// }

// sendMessageToUser replies to the chat the message came from, with the link preview disabled.
func (bot *TelegramBot) sendMessageToUser(chatID int64, text string) error {
	return bot.SendMessageToChat(fmt.Sprintf("%d", chatID), text, true)
}

// SendStartupNotification announces the bot startup in the service chat.
// A disabled bot is a no-op returning nil.
func (bot *TelegramBot) SendStartupNotification() error {
	if !bot.Enabled {
		return nil
	}

	return bot.sendMessage(TextStartupNotification)
}
