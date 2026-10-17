package db

import (
	"database/sql"
	"fmt"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/types"
)

// GetActiveSources возвращает все активные источники
func GetActiveSources() ([]types.Source, error) {
	database := GetDB()

	rows, err := database.Query(`
		SELECT id, parser_name, base_url, is_active, created_at, updated_at
		FROM sources
		WHERE is_active = true
	`)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса источников: %w", err)
	}
	defer rows.Close()

	var sources []types.Source
	for rows.Next() {
		var s types.Source
		err := rows.Scan(&s.ID, &s.ParserName, &s.BaseURL, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования источника: %w", err)
		}
		sources = append(sources, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации по источникам: %w", err)
	}

	return sources, nil
}

// GetSourceByName возвращает источник по имени парсера
func GetSourceByName(parserName types.SourceName) (*types.Source, error) {
	database := GetDB()

	var s types.Source
	err := database.QueryRow(`
		SELECT id, parser_name, base_url, is_active, created_at, updated_at
		FROM sources
		WHERE parser_name = $1
	`, parserName).Scan(&s.ID, &s.ParserName, &s.BaseURL, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса источника: %w", err)
	}

	return &s, nil
}

// GetSourceByBaseURL возвращает источник по базовому URL
// Поиск выполняется по точному совпадению или по вхождению хоста
func GetSourceByBaseURL(baseURL string) (*types.Source, error) {
	database := GetDB()

	var s types.Source

	// Сначала пробуем точное совпадение
	err := database.QueryRow(`
		SELECT id, parser_name, base_url, is_active, created_at, updated_at
		FROM sources
		WHERE base_url = $1 AND is_active = true
	`, baseURL).Scan(&s.ID, &s.ParserName, &s.BaseURL, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)

	if err == nil {
		return &s, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("ошибка запроса источника: %w", err)
	}

	// Если точного совпадения нет, ищем по LIKE (для случаев с/без https)
	err = database.QueryRow(`
		SELECT id, parser_name, base_url, is_active, created_at, updated_at
		FROM sources
		WHERE base_url LIKE '%' || $1 || '%' AND is_active = true
		LIMIT 1
	`, baseURL).Scan(&s.ID, &s.ParserName, &s.BaseURL, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса источника: %w", err)
	}

	return &s, nil
}
