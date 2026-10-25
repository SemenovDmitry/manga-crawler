package adminapi

import (
	"errors"
	"strings"
	"time"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/env"
)

type Config struct {
	// Addr — адрес HTTP-сервера (по умолчанию :8081).
	Addr string
	// Username — логин администратора.
	Username string
	// PasswordHash — bcrypt-хэш пароля администратора
	// (генерируется командой `manga-crawler-hashpassword`).
	PasswordHash string
	// JWTSecret — секрет подписи access-токенов (HS256). Обязателен.
	JWTSecret []byte
	// TokenTTL — срок жизни access-токена (по умолчанию 15 минут).
	TokenTTL time.Duration
	// RefreshTTL — срок жизни refresh-токена (по умолчанию 7 дней).
	RefreshTTL time.Duration
}

func ConfigFromEnv() Config {
	return Config{
		Addr:         env.String("ADMIN_LISTEN_ADDR", ":8081"),
		Username:     env.String("ADMIN_USERNAME", "admin"),
		PasswordHash: env.String("ADMIN_PASSWORD_HASH", ""),
		JWTSecret:    []byte(env.String("JWT_SECRET", "")),
		TokenTTL:     env.Duration("ADMIN_TOKEN_TTL", 15*time.Minute),
		RefreshTTL:   env.Duration("ADMIN_REFRESH_TTL", 7*24*time.Hour),
	}
}

func (c Config) Validate() error {
	if len(c.JWTSecret) == 0 {
		return errors.New("JWT_SECRET не задан — сгенерируйте: openssl rand -hex 32")
	}
	if len(c.JWTSecret) < 32 {
		return errors.New("JWT_SECRET короче 32 байт — небезопасно для HS256, сгенерируйте: openssl rand -hex 32")
	}
	if c.Username == "" {
		return errors.New("ADMIN_USERNAME не задан")
	}
	if c.PasswordHash == "" {
		return errors.New("ADMIN_PASSWORD_HASH не задан — сгенерируйте: go run ./cmd/hashpassword")
	}
	// Кривой хэш иначе выстрелит ошибкой bcrypt на каждый логин.
	if !strings.HasPrefix(c.PasswordHash, "$2a$") && !strings.HasPrefix(c.PasswordHash, "$2b$") && !strings.HasPrefix(c.PasswordHash, "$2y$") {
		return errors.New("ADMIN_PASSWORD_HASH не похож на bcrypt-хэш — перегенерируйте: go run ./cmd/hashpassword")
	}
	return nil
}
