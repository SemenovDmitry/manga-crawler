package db

import (
	"database/sql"
	"fmt"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/types"
)

// CreateSubscription создаёт подписку пользователя на мангу
func CreateSubscription(userID int64, mangaID int) (*types.UserSubscription, error) {
	database := GetDB()

	var sub types.UserSubscription
	err := database.QueryRow(`
		INSERT INTO user_subscriptions (user_id, manga_id, notify)
		VALUES ($1, $2, true)
		ON CONFLICT (user_id, manga_id) DO UPDATE SET notify = true
		RETURNING id, user_id, manga_id, notify, created_at
	`, userID, mangaID).Scan(&sub.ID, &sub.TelegramUserID, &sub.MangaID, &sub.Notify, &sub.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("ошибка создания подписки: %w", err)
	}

	return &sub, nil
}

// GetSubscription возвращает подписку пользователя на мангу
func GetSubscription(userID int64, mangaID int) (*types.UserSubscription, error) {
	database := GetDB()

	var sub types.UserSubscription
	err := database.QueryRow(`
		SELECT id, user_id, manga_id, notify, created_at
		FROM user_subscriptions
		WHERE user_id = $1 AND manga_id = $2
	`, userID, mangaID).Scan(&sub.ID, &sub.TelegramUserID, &sub.MangaID, &sub.Notify, &sub.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса подписки: %w", err)
	}

	return &sub, nil
}

// DeleteSubscription удаляет подписку пользователя на мангу
func DeleteSubscription(userID int64, mangaID int) error {
	database := GetDB()

	_, err := database.Exec(`
		DELETE FROM user_subscriptions
		WHERE user_id = $1 AND manga_id = $2
	`, userID, mangaID)

	if err != nil {
		return fmt.Errorf("ошибка удаления подписки: %w", err)
	}

	return nil
}

// MangaWithSource манга с информацией об источнике
type MangaWithSource struct {
	types.Manga
	SourceName    string `db:"source_name"`
	SourceBaseURL string `db:"source_base_url"`
}

// GetUserSubscriptions возвращает все подписки пользователя с информацией об источнике
func GetUserSubscriptions(userID int64) ([]MangaWithSource, error) {
	database := GetDB()

	rows, err := database.Query(`
		SELECT m.id, m.source_id, m.url, m.title, m.last_chapter_url, m.last_chapter_title, m.last_check_at, m.created_at, m.updated_at, s.parser_name, s.base_url
		FROM manga m
		JOIN user_subscriptions us ON m.id = us.manga_id
		JOIN sources s ON m.source_id = s.id
		WHERE us.user_id = $1 AND us.notify = true
		ORDER BY s.parser_name, m.title
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса подписок: %w", err)
	}
	defer rows.Close()

	var mangaList []MangaWithSource
	for rows.Next() {
		var m MangaWithSource
		var lastChapterURL, lastChapterTitle sql.NullString
		var lastCheckAt sql.NullTime

		err := rows.Scan(&m.ID, &m.SourceID, &m.URL, &m.Title, &lastChapterURL, &lastChapterTitle, &lastCheckAt, &m.CreatedAt, &m.UpdatedAt,
			&m.SourceName, &m.SourceBaseURL)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования манги: %w", err)
		}

		if lastChapterURL.Valid {
			m.LastChapterURL = lastChapterURL.String
		}
		if lastChapterTitle.Valid {
			m.LastChapterTitle = lastChapterTitle.String
		}
		if lastCheckAt.Valid {
			m.LastCheckAt = &lastCheckAt.Time
		}

		mangaList = append(mangaList, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации по подпискам: %w", err)
	}

	return mangaList, nil
}
