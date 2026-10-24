package storage

import (
	"fmt"
)

// Subscribe подписывает чат на мангу. Возвращает false, если подписка уже есть.
func (s *Store) Subscribe(chatID, mangaID int64) (bool, error) {
	const query = `INSERT INTO subscriptions (chat_id, manga_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`

	res, err := s.db.Exec(query, chatID, mangaID)
	if err != nil {
		return false, fmt.Errorf("ошибка подписки чата %d на мангу %d: %w", chatID, mangaID, err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("ошибка проверки подписки: %w", err)
	}

	return affected > 0, nil
}

// Unsubscribe отменяет подписку чата. Возвращает false, если подписки не было.
func (s *Store) Unsubscribe(chatID, mangaID int64) (bool, error) {
	const query = `DELETE FROM subscriptions WHERE chat_id = $1 AND manga_id = $2`

	res, err := s.db.Exec(query, chatID, mangaID)
	if err != nil {
		return false, fmt.Errorf("ошибка отписки чата %d от манги %d: %w", chatID, mangaID, err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("ошибка проверки отписки: %w", err)
	}

	return affected > 0, nil
}

// UserSubscriptions возвращает все подписки чата с данными о манге.
func (s *Store) UserSubscriptions(chatID int64) ([]Manga, error) {
	query := `SELECT ` + mangaColumns + `
		` + mangaFrom + `
		JOIN subscriptions sub ON sub.manga_id = m.id
		WHERE sub.chat_id = $1
		ORDER BY m.title`

	rows, err := s.db.Query(query, chatID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения подписок чата %d: %w", chatID, err)
	}
	defer rows.Close()

	var subscriptions []Manga
	for rows.Next() {
		manga, err := scanManga(rows)
		if err != nil {
			return nil, fmt.Errorf("ошибка чтения подписки чата %d: %w", chatID, err)
		}
		subscriptions = append(subscriptions, manga)
	}

	return subscriptions, rows.Err()
}

// SubscriberChatIDs возвращает чаты, подписанные на мангу.
func (s *Store) SubscriberChatIDs(mangaID int64) ([]int64, error) {
	const query = `SELECT chat_id FROM subscriptions WHERE manga_id = $1`

	rows, err := s.db.Query(query, mangaID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения подписчиков манги %d: %w", mangaID, err)
	}
	defer rows.Close()

	var chatIDs []int64
	for rows.Next() {
		var chatID int64
		if err := rows.Scan(&chatID); err != nil {
			return nil, fmt.Errorf("ошибка чтения подписчика манги %d: %w", mangaID, err)
		}
		chatIDs = append(chatIDs, chatID)
	}

	return chatIDs, rows.Err()
}
