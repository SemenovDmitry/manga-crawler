package storage

import (
	"database/sql"
	"fmt"
)

type ChatUser struct {
	ChatID    int64
	Username  string
	IsActive  bool
	CreatedAt sql.NullTime
	UpdatedAt sql.NullTime
}

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
