package telegram

import "strconv"

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
