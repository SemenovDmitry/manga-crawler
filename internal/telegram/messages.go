package telegram

import (
	"fmt"
	"log"
)

func (bot *Bot) sendMessageToChat(chatID int64, text string, disablePreview bool) error {
	return bot.sendMessageWithKeyboard(chatID, text, disablePreview, nil)
}

func (bot *Bot) sendMessageWithKeyboard(
	chatID int64,
	text string,
	disablePreview bool,
	keyboard *InlineKeyboardMarkup,
) error {
	message := MessageRequest{
		ChatID:                fmt.Sprintf("%d", chatID),
		Text:                  text,
		ParseMode:             "HTML",
		DisableWebPagePreview: disablePreview,
		ReplyMarkup:           keyboard,
	}

	if err := bot.callTelegram("sendMessage", message, nil); err != nil {
		return err
	}

	log.Printf("Telegram сообщение отправлено в чат %d", chatID)
	return nil
}

// editMessageText изменяет текст ранее отправленного ботом сообщения.
// Используется, чтобы заменять список результатов поиска подтверждением:
// старый текст и кнопки исчезают, и чат не засоряется неактуальными сообщениями.
// Telegram запрещает редактирование сообщений старше 48 часов — вызывающий
// код должен уметь fallback на отправку нового сообщения.
func (bot *Bot) editMessageText(
	chatID, messageID int64,
	text string,
	disablePreview bool,
	keyboard *InlineKeyboardMarkup,
) error {
	request := EditMessageTextRequest{
		ChatID:                fmt.Sprintf("%d", chatID),
		MessageID:             messageID,
		Text:                  text,
		ParseMode:             "HTML",
		DisableWebPagePreview: disablePreview,
		ReplyMarkup:           keyboard,
	}

	if err := bot.callTelegram("editMessageText", request, nil); err != nil {
		return err
	}

	log.Printf("Telegram сообщение %d в чате %d изменено", messageID, chatID)
	return nil
}

func (bot *Bot) answerCallbackQuery(callbackQueryID, text string) {
	request := AnswerCallbackQueryRequest{
		CallbackQueryID: callbackQueryID,
		Text:            text,
	}

	if err := bot.callTelegram("answerCallbackQuery", request, nil); err != nil {
		log.Printf("Ошибка ответа на кнопку: %v", err)
	}
}
