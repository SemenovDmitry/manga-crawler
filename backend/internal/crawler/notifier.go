package crawler

import (
	"fmt"
	"html"
	"log"
	"strings"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

// SourceInfo — минимальные данные источника для сообщений и логов
// (имя и базовый адрес), без зависимости от storage в сигнатурах.
type SourceInfo = storage.Source

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

// ServiceNotifier — доставка сервисных сообщений хозяину бота
// (автоотключение источника и т.п.). Реализуется QueueNotifier.
type ServiceNotifier interface {
	// ServiceMessage отправляет сообщение хозяину (TELEGRAM_CHAT_ID).
	ServiceMessage(text string)
}

// NotificationQueue — операции с очередью рассылки (реализуется
// *storage.Store): уведомления копятся в БД, рассылкой занимается
// процесс бота.
type NotificationQueue interface {
	SubscriberChatIDs(mangaID int64) ([]int64, error)
	EnqueueNotification(chatID, mangaID int64, text string) error
	EnqueueServiceNotification(chatID int64, text string) error
}

// QueueNotifier — реализация Notifier поверх очереди уведомлений в БД:
// формирует текст сообщения и ставит его каждому подписчику манги.
type QueueNotifier struct {
	queue       NotificationQueue
	ownerChatID int64 // 0 — сервисные сообщения не отправляются
}

// NewQueueNotifier собирает нотификатор над очередью рассылки.
// ownerChatID — чат хозяина для сервисных сообщений (0 — не отправлять).
func NewQueueNotifier(queue NotificationQueue, ownerChatID int64) *QueueNotifier {
	return &QueueNotifier{queue: queue, ownerChatID: ownerChatID}
}

// ServiceMessage ставит сервисное сообщение в очередь хозяину бота.
func (n *QueueNotifier) ServiceMessage(text string) {
	if n.ownerChatID == 0 {
		log.Printf("Сервисное сообщение не отправлено (TELEGRAM_CHAT_ID не задан): %s", singleLine(text))
		return
	}

	if err := n.queue.EnqueueServiceNotification(n.ownerChatID, text); err != nil {
		log.Printf("Ошибка постановки сервисного уведомления в очередь: %v", err)
	}
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

// formatSourceDisabledMessage собирает текст сервисного сообщения хозяину
// об автоотключении источника после серии ошибок парсинга.
func formatSourceDisabledMessage(src SourceInfo, failures int, parseErr error) string {
	return fmt.Sprintf(
		"🛑 <b>Источник «%s» отключён автоматически</b>\n"+
			"<i>Парсинг карточек падал %d раз подряд — похоже, сайт изменил разметку.</i>\n\n"+
			"Последняя ошибка: <code>%s</code>\n\n"+
			"Инкрементальный обход источника прерван, новые главы из него приходить не будут.\n"+
			"Включить обратно: админка → sources → enabled (после исправления причины).",
		html.EscapeString(src.Name),
		failures,
		html.EscapeString(truncateText(singleLine(errText(parseErr)), 300)),
	)
}

// errText безопасно извлекает текст ошибки (nil — заглушка).
func errText(err error) string {
	if err == nil {
		return "неизвестная ошибка"
	}
	return err.Error()
}

// singleLine сворачивает переносы в пробелы: сервисное сообщение уходит
// в одну строку лога, а внутри Telegram-текста переносы задаются явно.
func singleLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// truncateText ограничивает строку limit рунами, длинные — с многоточием.
func truncateText(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit]) + "…"
}
