package storage

import (
	"database/sql"
	"fmt"
)

// Source — источник парсинга (сайт с манги).
type Source struct {
	ID      int
	Name    string
	BaseURL string
	// FullCrawledAt — время последнего завершённого полного обхода;
	// NULL значит, что каталог источника ещё не загружен целиком.
	FullCrawledAt sql.NullTime
	// FullCrawlIntervalSecs — персональный интервал полного обхода в секундах
	// (по умолчанию 1209600 = 2 недели).
	FullCrawlIntervalSecs int64
	// NextFullCrawlAt — назначенное время следующего полного обхода;
	// NULL значит «обход нужен при первой возможности».
	NextFullCrawlAt sql.NullTime
	// Engine — имя движка сайта, по нему краулер выбирает драйвер
	// (grouple, mgeko, ...).
	Engine string
	// Enabled — источник участвует в обходах; false = сервис временно
	// выключен (краулер пропускает источник без рестарта).
	Enabled bool
}

const sourceColumns = `id, name, base_url, full_crawled_at, full_crawl_interval_secs, next_full_crawl_at, engine, enabled`

func scanSource(scanner interface{ Scan(dest ...any) error }) (Source, error) {
	var src Source
	if err := scanner.Scan(
		&src.ID, &src.Name, &src.BaseURL,
		&src.FullCrawledAt, &src.FullCrawlIntervalSecs, &src.NextFullCrawlAt,
		&src.Engine, &src.Enabled,
	); err != nil {
		return Source{}, err
	}
	return src, nil
}

// GetSources возвращает все источники, включая выключенные.
func (s *Store) GetSources() ([]Source, error) {
	rows, err := s.db.Query(`SELECT ` + sourceColumns + ` FROM sources ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения источников: %w", err)
	}
	defer rows.Close()

	var sources []Source
	for rows.Next() {
		src, err := scanSource(rows)
		if err != nil {
			return nil, fmt.Errorf("ошибка чтения источника: %w", err)
		}
		sources = append(sources, src)
	}

	return sources, rows.Err()
}

// GetEnabledSources возвращает только включённые источники (sources.enabled).
// Краулер перечитывает их перед каждым проходом, чтобы подхватить
// включение/выключение источника без рестарта.
func (s *Store) GetEnabledSources() ([]Source, error) {
	rows, err := s.db.Query(`SELECT ` + sourceColumns + ` FROM sources WHERE enabled ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения включённых источников: %w", err)
	}
	defer rows.Close()

	var sources []Source
	for rows.Next() {
		src, err := scanSource(rows)
		if err != nil {
			return nil, fmt.Errorf("ошибка чтения источника: %w", err)
		}
		sources = append(sources, src)
	}

	return sources, rows.Err()
}

// GetSourceByName возвращает источник по имени.
func (s *Store) GetSourceByName(name string) (Source, error) {
	query := `SELECT ` + sourceColumns + ` FROM sources WHERE name = $1`

	src, err := scanSource(s.db.QueryRow(query, name))
	if err != nil {
		return Source{}, fmt.Errorf("ошибка получения источника %s: %w", name, err)
	}

	return src, nil
}

// MarkSourceFullCrawled фиксирует завершённый полный обход источника и
// назначает следующий по его персональному интервалу. Возвращает назначенное
// время следующего обхода.
func (s *Store) MarkSourceFullCrawled(id int) (sql.NullTime, error) {
	const query = `
		UPDATE sources
		SET full_crawled_at = now(),
		    next_full_crawl_at = now() + make_interval(secs => full_crawl_interval_secs)
		WHERE id = $1
		RETURNING next_full_crawl_at`

	var next sql.NullTime
	if err := s.db.QueryRow(query, id).Scan(&next); err != nil {
		return sql.NullTime{}, fmt.Errorf("ошибка отметки полного обхода источника %d: %w", id, err)
	}

	return next, nil
}

// GetMangaCountBySource — количество карт в БД по каждому источнику (для логов краулера).
func (s *Store) GetMangaCountBySource() (map[string]int, error) {
	const query = `
		SELECT s.name, count(m.id)
		FROM sources s
		LEFT JOIN manga m ON m.parse_source_id = s.id
		GROUP BY s.name`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка подсчёта манг: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			return nil, fmt.Errorf("ошибка чтения счётчика манги: %w", err)
		}
		counts[name] = count
	}

	return counts, rows.Err()
}
