package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var ErrSourceNotFound = errors.New("источник не найден")

type SourceFilter struct {
	Enabled *bool
	Name    string
}

func EnabledOnly() SourceFilter {
	enabled := true
	return SourceFilter{Enabled: &enabled}
}

type Source struct {
	ID                    int
	Name                  string
	BaseURL               string
	FullCrawledAt         sql.NullTime
	FullCrawlIntervalSecs int64
	NextFullCrawlAt       sql.NullTime
	Engine                string
	Enabled               bool
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

func (s *Store) GetSources(filter SourceFilter) ([]Source, error) {
	query := `SELECT ` + sourceColumns + ` FROM sources`
	var conds []string
	var args []any
	if filter.Enabled != nil {
		args = append(args, *filter.Enabled)
		conds = append(conds, fmt.Sprintf("enabled = $%d", len(args)))
	}
	if filter.Name != "" {
		args = append(args, filter.Name)
		conds = append(conds, fmt.Sprintf("name = $%d", len(args)))
	}
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	query += " ORDER BY id"

	rows, err := s.db.Query(query, args...)
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

func (s *Store) GetSource(id int) (Source, error) {
	const query = `SELECT ` + sourceColumns + ` FROM sources WHERE id = $1`

	src, err := scanSource(s.db.QueryRow(query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Source{}, ErrSourceNotFound
	}
	if err != nil {
		return Source{}, fmt.Errorf("ошибка получения источника %d: %w", id, err)
	}
	return src, nil
}

func (s *Store) SetSourceEnabled(id int, enabled bool) (Source, error) {
	const query = `
		UPDATE sources SET enabled = $1
		WHERE id = $2
		RETURNING ` + sourceColumns

	src, err := scanSource(s.db.QueryRow(query, enabled, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Source{}, ErrSourceNotFound
	}
	if err != nil {
		return Source{}, fmt.Errorf("ошибка изменения источника %d: %w", id, err)
	}
	return src, nil
}

func (s *Store) ResetSourceFullCrawl(id int) (Source, error) {
	const query = `
		UPDATE sources SET next_full_crawl_at = NULL
		WHERE id = $1
		RETURNING ` + sourceColumns

	src, err := scanSource(s.db.QueryRow(query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Source{}, ErrSourceNotFound
	}
	if err != nil {
		return Source{}, fmt.Errorf("ошибка сброса расписания источника %d: %w", id, err)
	}
	return src, nil
}

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
