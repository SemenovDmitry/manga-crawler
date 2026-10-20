package telegram

import (
	"encoding/json"
	"fmt"
	"log"
)

func (bot *Bot) sendMessageToChat(chatID int64, text string, disablePreview bool) error {
	if !bot.Enabled {
		log.Println("Telegram бот отключен, сообщение не отправлено")
		return nil
	}

	message := MessageRequest{
		ChatID:                fmt.Sprintf("%d", chatID),
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

	log.Printf("Telegram сообщение отправлено в чат %d", chatID)
	return nil
}
