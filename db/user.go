package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/types"
)

// GetOrCreateUser получает или создаёт пользователя Telegram
func GetOrCreateUser(id int64, username, firstName, lastName string) (*types.TelegramUser, error) {
	database := GetDB()

	var u types.TelegramUser
	var usernameNull, firstNameNull, lastNameNull sql.NullString

	// Пробуем найти пользователя
	err := database.QueryRow(`
		SELECT id, username, first_name, last_name, is_active, created_at, updated_at
		FROM telegram_users
		WHERE id = $1
	`, id).Scan(&u.ID, &usernameNull, &firstNameNull, &lastNameNull, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)

	if err == nil {
		// Пользователь найден, обновляем данные если изменились
		if usernameNull.Valid {
			u.Username = usernameNull.String
		}
		if firstNameNull.Valid {
			u.FirstName = firstNameNull.String
		}
		if lastNameNull.Valid {
			u.LastName = lastNameNull.String
		}

		// Обновляем данные пользователя
		_, err = database.Exec(`
			UPDATE telegram_users
			SET username = $1, first_name = $2, last_name = $3, updated_at = $4
			WHERE id = $5
		`, nullString(username), nullString(firstName), nullString(lastName), time.Now(), id)

		if err != nil {
			return nil, fmt.Errorf("ошибка обновления пользователя: %w", err)
		}

		u.Username = username
		u.FirstName = firstName
		u.LastName = lastName

		return &u, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("ошибка запроса пользователя: %w", err)
	}

	// Пользователь не найден, создаём
	err = database.QueryRow(`
		INSERT INTO telegram_users (id, username, first_name, last_name, is_active)
		VALUES ($1, $2, $3, $4, true)
		RETURNING id, username, first_name, last_name, is_active, created_at, updated_at
	`, id, nullString(username), nullString(firstName), nullString(lastName)).Scan(
		&u.ID, &usernameNull, &firstNameNull, &lastNameNull, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("ошибка создания пользователя: %w", err)
	}

	if usernameNull.Valid {
		u.Username = usernameNull.String
	}
	if firstNameNull.Valid {
		u.FirstName = firstNameNull.String
	}
	if lastNameNull.Valid {
		u.LastName = lastNameNull.String
	}

	return &u, nil
}

// GetUserByID возвращает пользователя по ID
func GetUserByID(id int64) (*types.TelegramUser, error) {
	database := GetDB()

	var u types.TelegramUser
	var username, firstName, lastName sql.NullString

	err := database.QueryRow(`
		SELECT id, username, first_name, last_name, is_active, created_at, updated_at
		FROM telegram_users
		WHERE id = $1
	`, id).Scan(&u.ID, &username, &firstName, &lastName, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса пользователя: %w", err)
	}

	if username.Valid {
		u.Username = username.String
	}
	if firstName.Valid {
		u.FirstName = firstName.String
	}
	if lastName.Valid {
		u.LastName = lastName.String
	}

	return &u, nil
}

// nullString преобразует строку в sql.NullString
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
