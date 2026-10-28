package telegram

import (
	"log"
	"time"
)

const (
	dispatchBatchLimit   = 20
	dispatchPollInterval = 20 * time.Second
	dispatchRetryDelay   = 10 * time.Second
)

// StartNotificationDispatch читает очередь notifications и рассылает
// сообщения подписчикам. Вызывается в горутине из main.
func (bot *Bot) StartNotificationDispatch() {
	log.Println("Запуск рассылки уведомлений...")

	for {
		notifications, err := bot.Store.PendingNotifications(dispatchBatchLimit)
		if err != nil {
			log.Printf("Ошибка чтения очереди уведомлений: %v", err)
			time.Sleep(dispatchRetryDelay)
			continue
		}

		if len(notifications) == 0 {
			time.Sleep(dispatchPollInterval)
			continue
		}

		var sentIDs []int64
		for _, n := range notifications {
			if err := bot.sendMessageToChat(n.ChatID, n.Text, true); err != nil {
				if IsBotBlockedByUser(err) {
					// Пользователь заблокировал бота: is_active = FALSE
					// автоматически, уведомление снимается с очереди.
					// Когда пользователь разблокирует бота и напишет ему,
					// /start вернёт is_active = TRUE, и доставка возобновится.
					if markErr := bot.Store.MarkChatUserUnavailable(n.ChatID); markErr != nil {
						log.Printf("Ошибка отметки недоступности чата %d: %v", n.ChatID, markErr)
					}
					sentIDs = append(sentIDs, n.ID)
					log.Printf("Чат %d заблокировал бота, уведомление %d снято с очереди", n.ChatID, n.ID)
					continue
				}
				log.Printf("Ошибка отправки уведомления %d в чат %d: %v", n.ID, n.ChatID, err)
				continue // оставляем в очереди, попробуем в следующий раз
			}
			sentIDs = append(sentIDs, n.ID)
		}

		if err := bot.Store.MarkNotificationsSent(sentIDs); err != nil {
			log.Printf("Ошибка отметки отправленных уведомлений: %v", err)
		} else if len(sentIDs) > 0 {
			log.Printf("Отправлено уведомлений: %d", len(sentIDs))
		}
	}
}
