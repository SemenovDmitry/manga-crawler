package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type AdminSession struct {
	TokenHash string
	ExpiresAt time.Time
	CreatedAt sql.NullTime
	UsedAt    sql.NullTime
	Revoked   bool
}

var ErrSessionNotFound = errors.New("сессия не найдена")

func (s *Store) CreateAdminSession(sess AdminSession) error {
	const query = `
		INSERT INTO admin_sessions (token_hash, expires_at)
		VALUES ($1, $2)`

	if _, err := s.db.Exec(query, sess.TokenHash, sess.ExpiresAt); err != nil {
		return fmt.Errorf("ошибка создания сессии: %w", err)
	}
	return nil
}

func (s *Store) GetAdminSession(tokenHash string) (AdminSession, error) {
	const query = `
		SELECT token_hash, expires_at, created_at, used_at, revoked
		FROM admin_sessions
		WHERE token_hash = $1`

	var sess AdminSession
	err := s.db.QueryRow(query, tokenHash).Scan(
		&sess.TokenHash, &sess.ExpiresAt, &sess.CreatedAt, &sess.UsedAt, &sess.Revoked,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return AdminSession{}, ErrSessionNotFound
	}
	if err != nil {
		return AdminSession{}, fmt.Errorf("ошибка получения сессии: %w", err)
	}
	return sess, nil
}

func (s *Store) RotateAdminSession(oldHash string, new AdminSession) (bool, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("ошибка старта транзакции ротации сессии: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // при Commit безвреден

	// Гасим старый токен; затронуто 0 строк = токена нет / отозван / истёк.
	res, err := tx.Exec(`
		UPDATE admin_sessions
		SET revoked = TRUE, used_at = now()
		WHERE token_hash = $1 AND NOT revoked AND expires_at > now()`,
		oldHash,
	)
	if err != nil {
		return false, fmt.Errorf("ошибка отзыва старой сессии: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("ошибка подсчёта отозванных сессий: %w", err)
	}
	if affected == 0 {
		return false, nil
	}

	if _, err := tx.Exec(
		`INSERT INTO admin_sessions (token_hash, expires_at) VALUES ($1, $2)`,
		new.TokenHash, new.ExpiresAt,
	); err != nil {
		return false, fmt.Errorf("ошибка сохранения новой сессии: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("ошибка фиксации ротации сессии: %w", err)
	}
	return true, nil
}

func (s *Store) RevokeAdminSession(tokenHash string) error {
	const query = `
		UPDATE admin_sessions SET revoked = TRUE WHERE token_hash = $1`

	if _, err := s.db.Exec(query, tokenHash); err != nil {
		return fmt.Errorf("ошибка отзыва сессии: %w", err)
	}
	return nil
}

func (s *Store) DeleteExpiredAdminSessions() (int64, error) {
	res, err := s.db.Exec(`DELETE FROM admin_sessions WHERE expires_at < now()`)
	if err != nil {
		return 0, fmt.Errorf("ошибка удаления истёкших сессий: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("ошибка подсчёта удалённых сессий: %w", err)
	}
	return n, nil
}
