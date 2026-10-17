# manga-crawler-backend

Telegram-бот на Go для отслеживания обновлений манги.

## Команды бота

- `/start` — приветствие
- `/help` — справка

## Запуск

1. Скопируйте `.env.example` в `.env`, впишите `TELEGRAM_BOT_TOKEN` и `TELEGRAM_CHAT_ID`.
2. `go run ./cmd/bot` — или `docker compose up -d --build`.

Подробности: [infrastructure.md](infrastructure.md).
