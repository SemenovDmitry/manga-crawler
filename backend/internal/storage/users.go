package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var ErrChatUserNotFound = errors.New("пользователь чата не найден")

type ChatUserFilter struct {
	// Active — фильтр по is_active: true — только с доступом,
	// false — только без, nil — все.
	Active *bool
	// Blocked — фильтр по blocked: true — заблокировавшие бота,
	// false — не заблокировавшие, nil — все.
	Blocked *bool
}

type ChatUser struct {
	ChatID    int64
	Username  string
	IsActive  bool
	Blocked   bool
	CreatedAt sql.NullTime
	UpdatedAt sql.NullTime
}

const chatUserColumns = `chat_id, username, is_active, blocked, created_at, updated_at`

func scanChatUser(scanner interface{ Scan(dest ...any) error }) (ChatUser, error) {
	var u ChatUser
	if err := scanner.Scan(
		&u.ChatID, &u.Username, &u.IsActive, &u.Blocked, &u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		return ChatUser{}, err
	}
	return u, nil
}

func (s *Store) CreateChatUser(user ChatUser) (ChatUser, error) {
	const query = `
		INSERT INTO chat_users (chat_id, username, is_active, blocked)
		VALUES ($1, $2, $3, FALSE)
		ON CONFLICT (chat_id) DO UPDATE SET
			username   = EXCLUDED.username,
			is_active  = TRUE,
			updated_at = now()
		RETURNING ` + chatUserColumns

	var stored ChatUser
	if err := s.db.QueryRow(query,
		user.ChatID, user.Username, user.IsActive,
	).Scan(
		&stored.ChatID, &stored.Username, &stored.IsActive, &stored.Blocked,
		&stored.CreatedAt, &stored.UpdatedAt,
	); err != nil {
		return ChatUser{}, fmt.Errorf("ошибка сохранения пользователя чата: %w", err)
	}

	return stored, nil
}

func (s *Store) GetChatUser(chatID int64) (ChatUser, error) {
	const query = `SELECT ` + chatUserColumns + ` FROM chat_users WHERE chat_id = $1`

	user, err := scanChatUser(s.db.QueryRow(query, chatID))
	if errors.Is(err, sql.ErrNoRows) {
		return ChatUser{}, ErrChatUserNotFound
	}
	if err != nil {
		return ChatUser{}, fmt.Errorf("ошибка получения пользователя чата %d: %w", chatID, err)
	}

	return user, nil
}

func (s *Store) ListChatUsers(filter ChatUserFilter) ([]ChatUser, error) {
	query := `SELECT ` + chatUserColumns + ` FROM chat_users`

	var conds []string
	var args []any
	if filter.Active != nil {
		conds = append(conds, fmt.Sprintf("is_active = $%d", len(args)+1))
		args = append(args, *filter.Active)
	}
	if filter.Blocked != nil {
		conds = append(conds, fmt.Sprintf("blocked = $%d", len(args)+1))
		args = append(args, *filter.Blocked)
	}
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	query += " ORDER BY chat_id"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения пользователей чатов: %w", err)
	}
	defer rows.Close()

	var users []ChatUser
	for rows.Next() {
		u, err := scanChatUser(rows)
		if err != nil {
			return nil, fmt.Errorf("ошибка чтения пользователя чата: %w", err)
		}
		users = append(users, u)
	}

	return users, rows.Err()
}

func (s *Store) SetChatUserBlocked(chatID int64, blocked bool) (ChatUser, error) {
	const query = `
		UPDATE chat_users SET blocked = $1, updated_at = now()
		WHERE chat_id = $2
		RETURNING ` + chatUserColumns

	stored, err := scanChatUser(s.db.QueryRow(query, blocked, chatID))
	if errors.Is(err, sql.ErrNoRows) {
		return ChatUser{}, ErrChatUserNotFound
	}
	if err != nil {
		return ChatUser{}, fmt.Errorf("ошибка изменения блокировки чата %d: %w", chatID, err)
	}

	return stored, nil
}

func (s *Store) MarkChatUserUnavailable(chatID int64) error {
	const query = `
		UPDATE chat_users SET is_active = FALSE, updated_at = now()
		WHERE chat_id = $1`

	if _, err := s.db.Exec(query, chatID); err != nil {
		return fmt.Errorf("ошибка отметки недоступности чата %d: %w", chatID, err)
	}
	return nil
}
