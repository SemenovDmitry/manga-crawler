package db

import (
	"fmt"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/types"
)

// GetChaptersByMangaID возвращает все главы манги
func GetChaptersByMangaID(mangaID int) ([]types.Chapter, error) {
	database := GetDB()

	query := `
		SELECT id, manga_id, url, title, discovered_at
		FROM chapters
		WHERE manga_id = $1
		ORDER BY discovered_at DESC
	`

	rows, err := database.Query(query, mangaID)

	if err != nil {
		return nil, fmt.Errorf("ошибка запроса глав: %w", err)
	}

	defer rows.Close()

	var chapters []types.Chapter
	for rows.Next() {
		var c types.Chapter
		err := rows.Scan(&c.ID, &c.MangaID, &c.URL, &c.Title, &c.DiscoveredAt)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования главы: %w", err)
		}
		chapters = append(chapters, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации по главам: %w", err)
	}

	return chapters, nil
}

// CreateChapter создаёт новую главу
func CreateChapter(mangaID int, url, title string) (*types.Chapter, error) {
	database := GetDB()

	query := `
		INSERT INTO chapters (manga_id, url, title)
		VALUES ($1, $2, $3)
		ON CONFLICT (manga_id, url) DO NOTHING
		RETURNING id, manga_id, url, title, discovered_at
	`

	var c types.Chapter

	err := database.QueryRow(query, mangaID, url, title).Scan(&c.ID, &c.MangaID, &c.URL, &c.Title, &c.DiscoveredAt)

	if err != nil {
		// Если ON CONFLICT сработал, глава уже существует — не ошибка
		return nil, nil
	}

	return &c, nil
}

// CreateChapters создаёт несколько глав и возвращает только новые
func CreateChapters(mangaID int, chapters []types.Chapter) ([]types.Chapter, error) {
	var newChapters []types.Chapter

	for _, ch := range chapters {
		created, err := CreateChapter(mangaID, ch.URL, ch.Title)
		if err != nil {
			return nil, err
		}
		if created != nil {
			newChapters = append(newChapters, *created)
		}
	}

	return newChapters, nil
}

// ChapterExists проверяет существование главы
func ChapterExists(mangaID int, url string) (bool, error) {
	database := GetDB()

	var exists bool
	err := database.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM chapters WHERE manga_id = $1 AND url = $2)
	`, mangaID, url).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("ошибка проверки главы: %w", err)
	}

	return exists, nil
}
