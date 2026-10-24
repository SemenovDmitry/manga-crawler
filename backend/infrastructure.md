# Infrastructure — manga-crawler-backend

Проект — Telegram-бот на чистом Go (без веб-фреймворков и БД). Единственная внешняя
зависимость — `github.com/joho/godotenv` (чтение `.env`).

Как запустить (кратко — подробно в infrastructure.md)
  .env в корне: TELEGRAM_BOT_TOKEN=..., TELEGRAM_CHAT_ID=...
  docker compose up -d --build

go mod tidy - Вызывать после изменения зависимойстей перед билдом и коммитом
  синхронизирует go.mod и go.sum с реальным импортами кода: добавляет недостающие зависимости, убирает неиспользуемые, досчитывает хэши в go.sum.
