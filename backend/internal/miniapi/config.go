package miniapi

import (
	"errors"
	"time"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/env"
)

type Config struct {
	// Addr — адрес HTTP-сервера (по умолчанию :8082).
	Addr string
	// AllowedOrigin — CORS-origin мини-аппа (прод — https://SemenovDmitry.github.io).
	// Пусто — CORS-заголовки не ставятся: в dev фронт ходит через Vite-прокси,
	// это same-origin, предзапросы браузер не делает.
	AllowedOrigin string
	// BotToken — токен бота, из него выводится секрет проверки initData.
	// Обязателен (тот же TELEGRAM_BOT_TOKEN, что и у бота).
	BotToken string
	// InitDataMaxAge — порог свежести auth_date initData (по умолчанию 24 ч).
	InitDataMaxAge time.Duration
}

func ConfigFromEnv() Config {
	return Config{
		Addr:           env.String("MINIAPI_LISTEN", ":8082"),
		AllowedOrigin:  env.String("MINIAPI_ALLOWED_ORIGIN", ""),
		BotToken:       env.String("TELEGRAM_BOT_TOKEN", ""),
		InitDataMaxAge: env.Duration("MINIAPI_INITDATA_MAX_AGE", 24*time.Hour),
	}
}

func (c Config) Validate() error {
	if c.BotToken == "" {
		return errors.New("TELEGRAM_BOT_TOKEN не задан — без него невозможно проверять подпись initData")
	}
	if c.InitDataMaxAge <= 0 {
		return errors.New("MINIAPI_INITDATA_MAX_AGE должен быть положительным (например 24h)")
	}
	return nil
}
