package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/types"
)

// GetMangaBySourceID возвращает все манги для источника
func GetMangaBySourceID(sourceID int) ([]types.Manga, error) {
	database := GetDB()

	rows, err := database.Query(`
		SELECT id, source_id, url, title, last_chapter_url, last_chapter_title, last_check_at, created_at, updated_at
		FROM manga
		WHERE source_id = $1
	`, sourceID)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса манги: %w", err)
	}
	defer rows.Close()

	var mangaList []types.Manga
	for rows.Next() {
		var m types.Manga
		var lastChapterURL, lastChapterTitle sql.NullString
		var lastCheckAt sql.NullTime

		err := rows.Scan(&m.ID, &m.SourceID, &m.URL, &m.Title, &lastChapterURL, &lastChapterTitle, &lastCheckAt, &m.CreatedAt, &m.UpdatedAt)
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
		return nil, fmt.Errorf("ошибка итерации по манге: %w", err)
	}

	return mangaList, nil
}

// GetMangaByID возвращает мангу по ID
func GetMangaByID(id int) (*types.Manga, error) {
	database := GetDB()

	var m types.Manga
	var lastChapterURL, lastChapterTitle sql.NullString
	var lastCheckAt sql.NullTime

	err := database.QueryRow(`
		SELECT id, source_id, url, title, last_chapter_url, last_chapter_title, last_check_at, created_at, updated_at
		FROM manga
		WHERE id = $1
	`, id).Scan(&m.ID, &m.SourceID, &m.URL, &m.Title, &lastChapterURL, &lastChapterTitle, &lastCheckAt, &m.CreatedAt, &m.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса манги: %w", err)
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

	return &m, nil
}

// GetMangaBySourceAndURL возвращает мангу по source_id и url
func GetMangaBySourceAndURL(sourceID int, url string) (*types.Manga, error) {
	database := GetDB()

	var m types.Manga
	var lastChapterURL, lastChapterTitle sql.NullString
	var lastCheckAt sql.NullTime

	err := database.QueryRow(`
		SELECT id, source_id, url, title, last_chapter_url, last_chapter_title, last_check_at, created_at, updated_at
		FROM manga
		WHERE source_id = $1 AND url = $2
	`, sourceID, url).Scan(&m.ID, &m.SourceID, &m.URL, &m.Title, &lastChapterURL, &lastChapterTitle, &lastCheckAt, &m.CreatedAt, &m.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса манги: %w", err)
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

	return &m, nil
}

// UpdateMangaLastChapter обновляет информацию о последней главе
func UpdateMangaLastChapter(mangaID int, chapterURL, chapterTitle string) error {
	database := GetDB()

	_, err := database.Exec(`
		UPDATE manga
		SET last_chapter_url = $1, last_chapter_title = $2, last_check_at = $3, updated_at = $3
		WHERE id = $4
	`, chapterURL, chapterTitle, time.Now(), mangaID)

	if err != nil {
		return fmt.Errorf("ошибка обновления манги: %w", err)
	}

	return nil
}

// UpdateMangaLastCheck обновляет время последней проверки
func UpdateMangaLastCheck(mangaID int) error {
	database := GetDB()

	now := time.Now()
	_, err := database.Exec(`
		UPDATE manga
		SET last_check_at = $1, updated_at = $1
		WHERE id = $2
	`, now, mangaID)

	if err != nil {
		return fmt.Errorf("ошибка обновления времени проверки: %w", err)
	}

	return nil
}

// CreateManga создаёт новую мангу
func CreateManga(sourceID int, url, title string) (*types.Manga, error) {
	database := GetDB()

	var m types.Manga
	err := database.QueryRow(`
		INSERT INTO manga (source_id, url, title)
		VALUES ($1, $2, $3)
		RETURNING id, source_id, url, title, created_at, updated_at
	`, sourceID, url, title).Scan(&m.ID, &m.SourceID, &m.URL, &m.Title, &m.CreatedAt, &m.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("ошибка создания манги: %w", err)
	}

	return &m, nil
}

// GetMangaWithSubscribers возвращает мангу с подписчиками
func GetMangaWithSubscribers(mangaID int) (*types.Manga, error) {
	manga, err := GetMangaByID(mangaID)
	if err != nil || manga == nil {
		return manga, err
	}

	subscribers, err := GetMangaSubscribers(mangaID)
	if err != nil {
		return nil, err
	}

	manga.Subscribers = subscribers
	return manga, nil
}

// GetMangaSubscribers возвращает подписчиков манги
func GetMangaSubscribers(mangaID int) ([]types.TelegramUser, error) {
	database := GetDB()

	rows, err := database.Query(`
		SELECT tu.id, tu.username, tu.first_name, tu.last_name, tu.is_active, tu.created_at, tu.updated_at
		FROM telegram_users tu
		JOIN user_subscriptions us ON tu.id = us.user_id
		WHERE us.manga_id = $1 AND us.notify = true AND tu.is_active = true
	`, mangaID)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса подписчиков: %w", err)
	}
	defer rows.Close()

	var subscribers []types.TelegramUser
	for rows.Next() {
		var u types.TelegramUser
		var username, firstName, lastName sql.NullString

		err := rows.Scan(&u.ID, &username, &firstName, &lastName, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования подписчика: %w", err)
		}

		if username.Valid {
			u.Username = username.String
		}
		if firstName.Valid {
			u.FirstName = firstName.String
		}
		if lastName.Valid {
			u.LastName = lastName.String
		}

		subscribers = append(subscribers, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации по подписчикам: %w", err)
	}

	return subscribers, nil
}
