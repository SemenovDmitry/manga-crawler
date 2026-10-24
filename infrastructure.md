# Infrastructure — manga-crawler-backend

Проект — Telegram-бот на чистом Go (без веб-фреймворков и БД). Единственная внешняя
зависимость — `github.com/joho/godotenv` (чтение `.env`).

Как запустить (кратко — подробно в infrastructure.md)
  .env в корне: TELEGRAM_BOT_TOKEN=..., TELEGRAM_CHAT_ID=...
  docker compose up -d --build

go mod tidy - Вызывать после изменения зависимойстей перед билдом и коммитом
  синхронизирует go.mod и go.sum с реальным импортами кода: добавляет недостающие зависимости, убирает неиспользуемые, досчитывает хэши в go.sum.

Добавление нового источника
  Каждый сайт обслуживается драйвером — адаптером движка (internal/crawler, интерфейс SourceDriver:
  парсинг карточки + правила URL). Драйвер выбирается по sources.engine (internal/crawler/registry.go).
  1. Сайт на уже поддерживаемом движке (grouple, mgeko): только миграция —
     INSERT INTO sources (name, base_url, engine) VALUES (...);
  2. Новый движок: разведка (go run ./cmd/probe <url>, образцы в probe-output/<host>/),
     затем реализовать SourceDriver в internal/crawler/driver_<engine>.go,
     добавить case в registry.go и миграцию с INSERT'ом источника.
  Краулер (обходы, резюм после прерывания, уведомления) от движка не зависит.

