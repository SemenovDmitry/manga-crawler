package storage

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

// Notification — сообщение из очереди рассылки.
type Notification struct {
	ID        int64
	ChatID    int64
	MangaID   int64
	Text      string
	SentAt    sql.NullTime
	CreatedAt sql.NullTime
}

// EnqueueNotification добавляет сообщение в очередь рассылки.
func (s *Store) EnqueueNotification(chatID, mangaID int64, text string) error {
	const query = `INSERT INTO notifications (chat_id, manga_id, text) VALUES ($1, $2, $3)`

	if _, err := s.db.Exec(query, chatID, mangaID, text); err != nil {
		return fmt.Errorf("ошибка добавления уведомления в очередь: %w", err)
	}

	return nil
}

// PendingNotifications возвращает неотправленные уведомления.
func (s *Store) PendingNotifications(limit int) ([]Notification, error) {
	const query = `
		SELECT id, chat_id, manga_id, text
		FROM notifications
		WHERE sent_at IS NULL
		ORDER BY id
		LIMIT $1`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения уведомлений из очереди: %w", err)
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.ChatID, &n.MangaID, &n.Text); err != nil {
			return nil, fmt.Errorf("ошибка чтения уведомления из очереди: %w", err)
		}
		notifications = append(notifications, n)
	}

	return notifications, rows.Err()
}

// MarkNotificationsSent помечает уведомления отправленными.
func (s *Store) MarkNotificationsSent(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	const query = `UPDATE notifications SET sent_at = now() WHERE id = ANY($1)`

	if _, err := s.db.Exec(query, pq.Array(ids)); err != nil {
		return fmt.Errorf("ошибка отметки отправленных уведомлений: %w", err)
	}

	return nil
}
