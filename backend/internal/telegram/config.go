package telegram

import (
	"errors"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/env"
)

type Config struct {
	// Token — токен бота (TELEGRAM_BOT_TOKEN). Обязателен.
	Token string
	// OwnerChatID — чат хозяина для стартового уведомления
	// (TELEGRAM_CHAT_ID). Обязателен.
	OwnerChatID string
	// MiniAppURL — адрес мини-приложения для кнопки меню бота
	// (MINIAPP_URL). Пустая строка — кнопка меню не устанавливается.
	MiniAppURL string
}

func ConfigFromEnv() Config {
	return Config{
		Token:       env.String("TELEGRAM_BOT_TOKEN", ""),
		OwnerChatID: env.String("TELEGRAM_CHAT_ID", ""),
		MiniAppURL:  env.String("MINIAPP_URL", ""),
	}
}

func (c Config) Validate() error {
	if c.Token == "" {
		return errors.New("TELEGRAM_BOT_TOKEN не задан")
	}
	if c.OwnerChatID == "" {
		return errors.New("TELEGRAM_CHAT_ID не задан")
	}
	return nil
}
