package telegram

import "strconv"

// sendStartupNotification announces the bot startup in the service chat configured
// in TELEGRAM_CHAT_ID. A disabled bot is a no-op returning nil.
func (bot *Bot) sendStartupNotification() error {
	if !bot.Enabled {
		return nil
	}

	chatID, err := strconv.ParseInt(bot.ChatID, 10, 64)
	if err != nil {
		return err
	}

	return bot.sendMessageToChat(chatID, textStartupNotification, false)
}
