package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

var ErrMangaNotFound = errors.New("манга не найдена")

type Manga struct {
	ID                int64
	SourceID          int
	SourceName        string
	BaseURL           string
	URL               string // часть адреса, например "asura_2024"
	Title             string
	AltTitles         string
	LatestChapter     string
	ChaptersCount     int
	PublicationStatus PublicationStatus // см. константы Status* выше
	LastCheckAt       sql.NullTime
	CreatedAt         sql.NullTime
	UpdatedAt         sql.NullTime
}

type PublicationStatus string

const (
	StatusOngoing      PublicationStatus = "ongoing"
	StatusCompleted    PublicationStatus = "completed"
	StatusHiatus       PublicationStatus = "hiatus"
	StatusCancelled    PublicationStatus = "cancelled"
	StatusDiscontinued PublicationStatus = "discontinued"
)

// allPublicationStatuses — все допустимые значения статуса.
var allPublicationStatuses = []PublicationStatus{
	StatusOngoing, StatusCompleted, StatusHiatus, StatusCancelled, StatusDiscontinued,
}

// Valid сообщает, входит ли статус в множество допустимых значений.
func (s PublicationStatus) Valid() bool {
	for _, known := range allPublicationStatuses {
		if s == known {
			return true
		}
	}
	return false
}

func (s PublicationStatus) IsFinal() bool {
	switch s {
	case StatusCompleted, StatusCancelled, StatusDiscontinued:
		return true
	default:
		return false
	}
}

// finalPublicationStatusesSQL — финальные статусы в виде литерала
// текстового массива PostgreSQL (для `= ANY($n::text[])`).
func finalPublicationStatusesSQL() string {
	var final []string
	for _, s := range allPublicationStatuses {
		if s.IsFinal() {
			final = append(final, string(s))
		}
	}
	return "{" + strings.Join(final, ",") + "}"
}

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
	sourceID int, urlPart, title, altTitles, latestChapter string, publicationStatus PublicationStatus, chaptersCount int,
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

func (s *Store) GetManga(id int64) (Manga, error) {
	query := `SELECT ` + mangaColumns + ` ` + mangaFrom + ` WHERE m.id = $1`

	manga, err := scanManga(s.db.QueryRow(query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Manga{}, ErrMangaNotFound
	}
	if err != nil {
		return Manga{}, fmt.Errorf("ошибка получения манги %d: %w", id, err)
	}

	return manga, nil
}

type MangaFilter struct {
	SourceID          int
	Query             string
	PublicationStatus PublicationStatus // пустое значение — фильтр не применяется
	Limit             int
	Offset            int
}

func (s *Store) ListManga(filter MangaFilter) ([]Manga, error) {
	query := `SELECT ` + mangaColumns + ` ` + mangaFrom

	var conds []string
	var args []any
	if filter.SourceID != 0 {
		args = append(args, filter.SourceID)
		conds = append(conds, fmt.Sprintf("m.parse_source_id = $%d", len(args)))
	}
	if filter.Query != "" {
		args = append(args, "%"+escapeLike(strings.ToLower(filter.Query))+"%")
		conds = append(conds, fmt.Sprintf(
			"(lower(m.title) LIKE $%d ESCAPE '\\' OR lower(m.alt_titles) LIKE $%d ESCAPE '\\')",
			len(args), len(args),
		))
	}
	if filter.PublicationStatus != "" {
		args = append(args, filter.PublicationStatus)
		conds = append(conds, fmt.Sprintf("m.publication_status = $%d", len(args)))
	}
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	query += " ORDER BY m.id"

	if filter.Limit > 0 {
		args = append(args, filter.Limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}
	if filter.Offset > 0 {
		args = append(args, filter.Offset)
		query += fmt.Sprintf(" OFFSET $%d", len(args))
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения манг: %w", err)
	}
	defer rows.Close()

	var results []Manga
	for rows.Next() {
		m, err := scanManga(rows)
		if err != nil {
			return nil, fmt.Errorf("ошибка чтения манги: %w", err)
		}
		results = append(results, m)
	}

	return results, rows.Err()
}

func (s *Store) CreateManga(m Manga) (Manga, error) {
	const query = `
		INSERT INTO manga (parse_source_id, url, title, alt_titles, latest_chapter, chapters_count, publication_status, last_check_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now())
		RETURNING ` + mangaColumns

	created, err := scanManga(s.db.QueryRow(query,
		m.SourceID, m.URL, m.Title, m.AltTitles, m.LatestChapter, m.ChaptersCount, m.PublicationStatus,
	))
	if err != nil {
		// Нарушение FK по parse_source_id — источника с таким id нет.
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			return Manga{}, ErrSourceNotFound
		}
		return Manga{}, fmt.Errorf("ошибка создания манги: %w", err)
	}

	return created, nil
}

type MangaPatch struct {
	Title             *string
	AltTitles         *string
	LatestChapter     *string
	ChaptersCount     *int
	PublicationStatus *PublicationStatus
}

func (s *Store) UpdateManga(id int64, patch MangaPatch) (Manga, error) {
	var sets []string
	var args []any
	args = append(args, id) // $1 — id в WHERE

	if patch.Title != nil {
		args = append(args, *patch.Title)
		sets = append(sets, fmt.Sprintf("title = $%d", len(args)))
	}
	if patch.AltTitles != nil {
		args = append(args, *patch.AltTitles)
		sets = append(sets, fmt.Sprintf("alt_titles = $%d", len(args)))
	}
	if patch.LatestChapter != nil {
		args = append(args, *patch.LatestChapter)
		sets = append(sets, fmt.Sprintf("latest_chapter = $%d", len(args)))
	}
	if patch.ChaptersCount != nil {
		args = append(args, *patch.ChaptersCount)
		sets = append(sets, fmt.Sprintf("chapters_count = $%d", len(args)))
	}
	if patch.PublicationStatus != nil {
		args = append(args, *patch.PublicationStatus)
		sets = append(sets, fmt.Sprintf("publication_status = $%d", len(args)))
	}

	if len(sets) == 0 {
		return s.GetManga(id)
	}

	query := `
		UPDATE manga SET
			` + strings.Join(sets, ", ") + `,
			updated_at         = now()
		WHERE id = $1
		RETURNING ` + mangaColumns

	updated, err := scanManga(s.db.QueryRow(query, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return Manga{}, ErrMangaNotFound
	}
	if err != nil {
		return Manga{}, fmt.Errorf("ошибка обновления манги %d: %w", id, err)
	}

	return updated, nil
}

func (s *Store) DeleteManga(id int64) (Manga, error) {
	const query = `DELETE FROM manga WHERE id = $1 RETURNING ` + mangaColumns

	deleted, err := scanManga(s.db.QueryRow(query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Manga{}, ErrMangaNotFound
	}
	if err != nil {
		return Manga{}, fmt.Errorf("ошибка удаления манги %d: %w", id, err)
	}

	return deleted, nil
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
// Завершённые и прекращённые манги исключаются: новых глав у них
// точно не будет, даже если подписка остаётся. Hiatus не исключается:
// это перерыв без даты окончания, публикация может возобновиться.
func (s *Store) GetStaleSubscribedManga(sourceID, staleSeconds, limit int) ([]Manga, error) {
	excluded := finalPublicationStatusesSQL()

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
