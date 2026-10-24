package crawler

import (
	"fmt"
	"html"
	"log"
	"strings"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers"
)

// maxNotificationChapters — сколько номеров глав показывать в уведомлении;
// остальное сворачивается в «(и ещё N)».
const maxNotificationChapters = 5

// Notifier — постановка уведомлений о новых главах подписчикам. Краулер
// зависит только от этого контракта: кто, куда и в каком виде доставит
// уведомление — вне его ответственности.
type Notifier interface {
	// NewChapters вызывается, когда у известной манги нашлись новые главы.
	NewChapters(mangaID int64, rawURL, title string, newChapters []parsers.Chapter)
}

// NotificationQueue — операции с очередью рассылки (реализуется
// *storage.Store): уведомления копятся в БД, рассылкой занимается
// процесс бота.
type NotificationQueue interface {
	SubscriberChatIDs(mangaID int64) ([]int64, error)
	EnqueueNotification(chatID, mangaID int64, text string) error
}

// QueueNotifier — реализация Notifier поверх очереди уведомлений в БД:
// формирует текст сообщения и ставит его каждому подписчику манги.
type QueueNotifier struct {
	queue NotificationQueue
}

// NewQueueNotifier собирает нотификатор над очередью рассылки.
func NewQueueNotifier(queue NotificationQueue) *QueueNotifier {
	return &QueueNotifier{queue: queue}
}

// NewChapters ставит уведомление о новых главах в очередь рассылки
// для каждого подписчика манги.
func (n *QueueNotifier) NewChapters(mangaID int64, rawURL, title string, newChapters []parsers.Chapter) {
	chatIDs, err := n.queue.SubscriberChatIDs(mangaID)
	if err != nil {
		log.Printf("Ошибка получения подписчиков манги %d: %v", mangaID, err)
		return
	}
	if len(chatIDs) == 0 {
		return
	}

	text := formatNewChaptersMessage(title, rawURL, newChapters)

	for _, chatID := range chatIDs {
		if err := n.queue.EnqueueNotification(chatID, mangaID, text); err != nil {
			log.Printf("Ошибка постановки уведомления в очередь (чат %d): %v", chatID, err)
		}
	}

	log.Printf("Манга %q (%d): новых глав %d, подписчиков %d", title, mangaID, len(newChapters), len(chatIDs))
}

// formatNewChaptersMessage собирает текст уведомления (Telegram HTML):
// заголовок, до maxNotificationChapters номеров глав и ссылка на чтение.
func formatNewChaptersMessage(title, rawURL string, newChapters []parsers.Chapter) string {
	shown := newChapters
	suffix := ""
	if len(shown) > maxNotificationChapters {
		shown = shown[:maxNotificationChapters]
		suffix = fmt.Sprintf(" (и ещё %d)", len(newChapters)-maxNotificationChapters)
	}

	numbers := make([]string, 0, len(shown))
	for _, ch := range shown {
		numbers = append(numbers, ch.Number)
	}

	return fmt.Sprintf(
		"📚 <b>%s</b> — новые главы!\n\n%s%s\n\n<a href=\"%s\">Читать</a>",
		html.EscapeString(title),
		html.EscapeString(strings.Join(numbers, ", ")),
		suffix,
		rawURL,
	)
}
