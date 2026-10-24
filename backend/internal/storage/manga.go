package storage

import (
	"database/sql"
	"fmt"
	"strings"
)

// Manga — запись о манге.
type Manga struct {
	ID            int64
	SourceID      int
	SourceName    string
	BaseURL       string
	URL           string // часть адреса, например "asura_2024"
	Title         string
	AltTitles     string
	LatestChapter string
	ChaptersCount int
	// PublicationStatus — ongoing, completed, hiatus, cancelled, discontinued.
	PublicationStatus string
	LastCheckAt       sql.NullTime
	CreatedAt         sql.NullTime
	UpdatedAt         sql.NullTime
}

// Статусы публикации, при которых манга не перепарсируется инкрементально:
// главы уже не выходят (перевод завершён/приостановлен и т.п.).
var finalPublicationStatuses = []string{"completed", "hiatus", "cancelled", "discontinued"}

// MangaState — состояние манги при прошлом проходе (для детекта новых глав).
type MangaState struct {
	MangaID       int64
	LatestChapter string
	ChaptersCount int
}

const mangaColumns = `m.id, m.parse_source_id, s.name, s.base_url, m.url, m.title,
	m.alt_titles, m.latest_chapter, m.chapters_count, m.publication_status,
	m.last_check_at, m.created_at, m.updated_at`

const mangaFrom = `FROM manga m JOIN sources s ON s.id = m.parse_source_id`

func scanManga(scanner interface{ Scan(dest ...any) error }) (Manga, error) {
	var m Manga
	if err := scanner.Scan(
		&m.ID, &m.SourceID, &m.SourceName, &m.BaseURL, &m.URL, &m.Title,
		&m.AltTitles, &m.LatestChapter, &m.ChaptersCount, &m.PublicationStatus,
		&m.LastCheckAt, &m.CreatedAt, &m.UpdatedAt,
	); err != nil {
		return Manga{}, err
	}
	return m, nil
}

// escapeLike экранирует спецсимволы LIKE в пользовательском запросе.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// GetMangaState возвращает сохранённое состояние манги (found=false, если манги ещё нет в БД).
func (s *Store) GetMangaState(sourceID int, urlPart string) (MangaState, bool, error) {
	const query = `SELECT id, latest_chapter, chapters_count FROM manga WHERE parse_source_id = $1 AND url = $2`

	var state MangaState
	err := s.db.QueryRow(query, sourceID, urlPart).
		Scan(&state.MangaID, &state.LatestChapter, &state.ChaptersCount)
	if err == sql.ErrNoRows {
		return MangaState{}, false, nil
	}
	if err != nil {
		return MangaState{}, false, fmt.Errorf("ошибка получения состояния манги %s: %w", urlPart, err)
	}

	return state, true, nil
}

// UpsertManga создаёт запись манги или обновляет её данные, возвращает ID.
// Пустой publicationStatus означает «статус не определён»: при вставке
// используется ongoing, при обновлении прежнее значение сохраняется —
// чтобы случайная правка вёрстки не «оживляла» завершённую мангу.
func (s *Store) UpsertManga(
	sourceID int, urlPart, title, altTitles, latestChapter, publicationStatus string, chaptersCount int,
) (int64, error) {
	const query = `
		INSERT INTO manga (parse_source_id, url, title, alt_titles, latest_chapter, publication_status, chapters_count, last_check_at)
		VALUES ($1, $2, $3, $4, $5, COALESCE(NULLIF($6, ''), 'ongoing'), $7, now())
		ON CONFLICT (parse_source_id, url) DO UPDATE SET
			title              = EXCLUDED.title,
			alt_titles         = EXCLUDED.alt_titles,
			latest_chapter     = EXCLUDED.latest_chapter,
			publication_status = CASE
				WHEN EXCLUDED.publication_status <> 'ongoing' THEN EXCLUDED.publication_status
				ELSE manga.publication_status END,
			chapters_count     = EXCLUDED.chapters_count,
			last_check_at      = now(),
			updated_at         = now()
		RETURNING id`

	var id int64
	if err := s.db.QueryRow(query,
		sourceID, urlPart, title, altTitles, latestChapter, publicationStatus, chaptersCount,
	).Scan(&id); err != nil {
		return 0, fmt.Errorf("ошибка сохранения манги %s: %w", urlPart, err)
	}

	return id, nil
}

// GetManga возвращает мангу по ID.
func (s *Store) GetManga(id int64) (Manga, error) {
	query := `SELECT ` + mangaColumns + ` ` + mangaFrom + ` WHERE m.id = $1`

	manga, err := scanManga(s.db.QueryRow(query, id))
	if err != nil {
		return Manga{}, fmt.Errorf("ошибка получения манги %d: %w", id, err)
	}

	return manga, nil
}

// GetMangaURLs возвращает все url-части манг источника (для диффа sitemap).
func (s *Store) GetMangaURLs(sourceID int) ([]string, error) {
	const query = `SELECT url FROM manga WHERE parse_source_id = $1`

	rows, err := s.db.Query(query, sourceID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения url манг источника %d: %w", sourceID, err)
	}
	defer rows.Close()

	var urls []string
	for rows.Next() {
		var urlPart string
		if err := rows.Scan(&urlPart); err != nil {
			return nil, fmt.Errorf("ошибка чтения url манги источника %d: %w", sourceID, err)
		}
		urls = append(urls, urlPart)
	}

	return urls, rows.Err()
}

// GetStaleSubscribedManga возвращает мангу источника с подписчиками,
// которая не проверялась дольше staleSeconds (для инкрементального прохода).
// Завершённые, приостановленные и прекращённые манги исключаются: новых
// глав у них не будет, даже если подписка остаётся.
func (s *Store) GetStaleSubscribedManga(sourceID, staleSeconds, limit int) ([]Manga, error) {
	excluded := "{" + strings.Join(finalPublicationStatuses, ",") + "}"

	query := `SELECT ` + mangaColumns + `
		` + mangaFrom + `
		JOIN subscriptions sub ON sub.manga_id = m.id
		WHERE m.parse_source_id = $1
		  AND (m.last_check_at IS NULL OR m.last_check_at < now() - make_interval(secs => $2::float8))
		  AND NOT (m.publication_status = ANY($3::text[]))
		LIMIT $4`

	rows, err := s.db.Query(query, sourceID, staleSeconds, excluded, limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения устаревших подписок источника %d: %w", sourceID, err)
	}
	defer rows.Close()

	var mangaList []Manga
	for rows.Next() {
		manga, err := scanManga(rows)
		if err != nil {
			return nil, fmt.Errorf("ошибка чтения устаревшей подписки источника %d: %w", sourceID, err)
		}
		mangaList = append(mangaList, manga)
	}

	return mangaList, rows.Err()
}

// SearchManga ищет по названиям (title и alt_titles), без учёта регистра.
func (s *Store) SearchManga(query string, limit int) ([]Manga, error) {
	pattern := "%" + escapeLike(strings.ToLower(strings.TrimSpace(query))) + "%"
	searchQuery := `SELECT ` + mangaColumns + ` ` + mangaFrom + `
		WHERE lower(m.title) LIKE $1 ESCAPE '\'
		   OR lower(m.alt_titles) LIKE $1 ESCAPE '\'
		ORDER BY m.chapters_count DESC, m.title
		LIMIT $2`

	rows, err := s.db.Query(searchQuery, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка поиска манги %q: %w", query, err)
	}
	defer rows.Close()

	var results []Manga
	for rows.Next() {
		manga, err := scanManga(rows)
		if err != nil {
			return nil, fmt.Errorf("ошибка чтения результата поиска: %w", err)
		}
		results = append(results, manga)
	}

	return results, rows.Err()
}
