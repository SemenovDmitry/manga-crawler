# Обзор проекта manga-crawler-backend

## Что это

Telegram-бот на чистом Go для отслеживания обновлений манги. 

## Стек

- Go (в `go.mod` — 1.26), без фреймворков.
- PostgreSQL 16 (в docker-compose — `postgres:16-alpine`), драйвер `github.com/lib/pq`,
  миграции — `github.com/pressly/goose/v3`.

## Структура

- `cmd/bot/main.go` — входная точка: подключает store, пингует БД, гоняет миграции,
  запускает polling в горутине и блокирует `main` через `select {}`.
- `internal/storage/` — доступ к БД (`storage.go`, `users.go`), миграции (`migrations.go`)
  и SQL-файлы миграций в `internal/storage/migrations/` (`00001_init.sql`,
  `00002_add_chat_users.sql`). Новые миграции добавлять туда же по порядку номеров.
- `internal/telegram/` — вся работа с Telegram: `client.go`, `handlers.go`, `messages.go`,
  `texts.go`, `types.go`. Тексты сообщений собраны в `texts.go` / `messages.go`.
- `internal/types/` — доменные типы и парсинг XML/RSS (`domain.go`, `types.go`, `xml.go`).

## Запуск и окружение

1. `.env` в корне (шаблон — `.env.example`): `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID`,
   `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`.
2. Локально: `go run ./cmd/bot` (в Makefile есть `server-start` и `build`).
3. В Docker: `docker compose up -d --build` — поднимает БД с healthcheck, потом бота
   (порт 8080, `DB_HOST=db`).
4. После изменения зависимостей — `go mod tidy` перед билдом и коммитом.
