package telegram

import (
	"context"
	"log"
	"time"
)

const (
	dispatchBatchLimit   = 20
	dispatchPollInterval = 20 * time.Second
	dispatchRetryDelay   = 10 * time.Second
)

// StartNotificationDispatch читает очередь notifications и рассылает
// сообщения подписчикам, пока контекст жив. Вызывается в горутине из main.
func (bot *Bot) StartNotificationDispatch(ctx context.Context) {
	log.Println("Запуск рассылки уведомлений...")

	for ctx.Err() == nil {
		bot.dispatchBatch(ctx)
	}

	log.Println("Рассылка уведомлений остановлена")
}

// dispatchBatch разбирает одну порцию очереди: отправка сообщений, отметка
// отправленных, снятие уведомлений чатов, заблокировавших бота.
// Недоставленное остаётся в очереди до следующей порции.
func (bot *Bot) dispatchBatch(ctx context.Context) {
	notifications, err := bot.backend.PendingNotifications(dispatchBatchLimit)
	if err != nil {
		log.Printf("Ошибка чтения очереди уведомлений: %v", err)
		time.Sleep(dispatchRetryDelay)
		return
	}

	if len(notifications) == 0 {
		time.Sleep(dispatchPollInterval)
		return
	}

	var sentIDs []int64
	for _, n := range notifications {
		// Остановка между сообщениями: уже отправленное отмечается
		// в очереди, недоставленное остаётся до следующего запуска.
		if ctx.Err() != nil {
			break
		}
		if err := bot.sendMessageToChat(n.ChatID, n.Text, true); err != nil {
			if IsBotBlockedByUser(err) {
				// Пользователь заблокировал бота: is_active = FALSE
				// автоматически, уведомление снимается с очереди.
				// Когда пользователь разблокирует бота и напишет ему,
				// /start вернёт is_active = TRUE, и доставка возобновится.
				if markErr := bot.backend.MarkChatUserUnavailable(n.ChatID); markErr != nil {
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

	if err := bot.backend.MarkNotificationsSent(sentIDs); err != nil {
		log.Printf("Ошибка отметки отправленных уведомлений: %v", err)
	} else if len(sentIDs) > 0 {
		log.Printf("Отправлено уведомлений: %d", len(sentIDs))
	}
}
