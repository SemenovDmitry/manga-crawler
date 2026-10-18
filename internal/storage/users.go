package storage

import (
	"database/sql"
	"fmt"
)

// ChatUser is a chat record where the bot was started, storing the profile
// of the user who activated it.
type ChatUser struct {
	ChatID    int64
	Username  string
	IsActive  bool
	CreatedAt sql.NullTime
	UpdatedAt sql.NullTime
}

// CreateChatUser inserts the chat record or refreshes the activating user's
// profile on conflict, so repeated /start commands keep the record up to date.
// An existing chat is reactivated instead of being overwritten with a new flag,
// and the stored row (including DB-generated timestamps) is returned.
func (s *Store) CreateChatUser(user ChatUser) (ChatUser, error) {
	const query = `
		INSERT INTO chat_users (chat_id, username, is_active)
		VALUES ($1, $2, $3)
		ON CONFLICT (chat_id) DO UPDATE SET
			username   = EXCLUDED.username,
			is_active  = TRUE,
			updated_at = now()
		RETURNING chat_id, username, is_active, created_at, updated_at`

	var stored ChatUser
	if err := s.db.QueryRow(query,
		user.ChatID, user.Username, user.IsActive,
	).Scan(
		&stored.ChatID, &stored.Username, &stored.IsActive,
		&stored.CreatedAt, &stored.UpdatedAt,
	); err != nil {
		return ChatUser{}, fmt.Errorf("ошибка сохранения пользователя чата: %w", err)
	}

	return stored, nil
}

// GetChatUser returns the chat record by the chat's Telegram id;
// the error wraps sql.ErrNoRows when the chat is not found.
func (s *Store) GetChatUser(chatID int64) (ChatUser, error) {
	const query = `
		SELECT chat_id, username, is_active, created_at, updated_at
		FROM chat_users
		WHERE chat_id = $1`

	var user ChatUser
	if err := s.db.QueryRow(query, chatID).Scan(
		&user.ChatID, &user.Username, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt,
	); err != nil {
		return ChatUser{}, fmt.Errorf("ошибка получения пользователя чата %d: %w", chatID, err)
	}

	return user, nil
}
