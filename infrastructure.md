# Infrastructure Guide

## Docker Compose

### Структура сервисов

```
manga-crawler-backend     # Основное приложение
manga-crawler-db          # PostgreSQL база данных
```

---

## Быстрый старт

```bash
# 1. Скопировать конфиг
cp .env.example .env

# 2. Заполнить переменные в .env
# TELEGRAM_BOT_TOKEN=...
# TELEGRAM_CHAT_ID=...

# 3. Запустить всё
docker-compose up -d --build
```

---

## Команды Docker Compose

### Сборка и запуск

```bash
# Запустить все сервисы
docker-compose up -d

# Пересобрать и запустить (после изменений в коде)
docker-compose up -d --build

# Пересобрать с нуля (без кэша)
docker-compose build --no-cache
docker-compose up -d

# Пересобрать только backend
docker-compose up -d --build manga-crawler-backend
```

### Остановка и удаление

```bash
# Остановить все сервисы
docker-compose stop

# Остановить и удалить контейнеры
docker-compose down

# Удалить всё включая volumes (ВНИМАНИЕ: удалит данные БД!)
docker-compose down -v

# Удалить orphan контейнеры
docker-compose up -d --remove-orphans
```

### Логи

```bash
# Все логи
docker-compose logs

# Логи в реальном времени
docker-compose logs -f

# Логи конкретного сервиса
docker-compose logs -f manga-crawler-backend
docker-compose logs -f manga-crawler-db

# Последние 100 строк
docker-compose logs --tail=100 manga-crawler-backend
```

### Статус

```bash
# Список контейнеров
docker-compose ps

# Статистика ресурсов
docker stats
```

---

## База данных

### Подключение к PostgreSQL

```bash
# Подключиться к БД
docker exec -it manga-crawler-db psql -U mangauser -d mangadb

# Выполнить SQL команду
docker exec -it manga-crawler-db psql -U mangauser -d mangadb -c "SELECT * FROM sources;"
```

### Полезные SQL команды

```sql
-- Список таблиц
\dt

-- Структура таблицы
\d sources
\d manga

-- Посмотреть источники
SELECT * FROM sources;

-- Посмотреть мангу
SELECT id, title, url FROM manga;

-- Посмотреть подписчиков
SELECT tu.username, m.title 
FROM user_subscriptions us
JOIN telegram_users tu ON tu.id = us.user_id
JOIN manga m ON m.id = us.manga_id;

-- Выход
\q
```

### Сброс базы данных

```bash
# Полный сброс (удаление volume)
docker-compose down -v
docker-compose up -d

# Или очистить таблицы вручную
docker exec -it manga-crawler-db psql -U mangauser -d mangadb -c "
DROP TABLE IF EXISTS chapters CASCADE;
DROP TABLE IF EXISTS user_subscriptions CASCADE;
DROP TABLE IF EXISTS manga CASCADE;
DROP TABLE IF EXISTS sources CASCADE;
DROP TABLE IF EXISTS telegram_users CASCADE;
DROP TABLE IF EXISTS goose_db_version CASCADE;
"
```

---

## Миграции

Миграции запускаются **автоматически** при старте приложения через goose.

### Ручной запуск миграций

```bash
# Установить goose (если нужно локально)
go install github.com/pressly/goose/v3/cmd/goose@latest

# Применить миграции
goose -dir storage/migrations postgres "host=localhost port=5432 user=mangauser password=mangapass dbname=mangadb sslmode=disable" up

# Откатить последнюю миграцию
goose -dir storage/migrations postgres "host=localhost port=5432 user=mangauser password=mangapass dbname=mangadb sslmode=disable" down

# Статус миграций
goose -dir storage/migrations postgres "host=localhost port=5432 user=mangauser password=mangapass dbname=mangadb sslmode=disable" status
```

---

## Пересборка проекта

### После изменений в коде

```bash
docker-compose up -d --build
```

### Полная пересборка с очисткой

```bash
# Остановить и удалить контейнеры (сохранить данные)
docker-compose down

# Удалить старый образ
docker rmi manga-crawler-backend-manga-crawler-backend

# Пересобрать и запустить
docker-compose up -d --build
```

### Пересборка с очисткой БД

```bash
# Остановить, удалить контейнеры и volumes
docker-compose down -v

# Пересобрать и запустить (БД создастся заново)
docker-compose up -d --build
```

---

## Разработка

### Локальный запуск (без Docker)

```bash
# 1. Запустить только БД
docker-compose up -d manga-crawler-db

# 2. Запустить приложение локально
go run main.go
```

### Проверка работы

```bash
# Проверить что контейнеры запущены
docker-compose ps

# Проверить логи на ошибки
docker-compose logs -f manga-crawler-backend

# Проверить что БД инициализирована
docker exec -it manga-crawler-db psql -U mangauser -d mangadb -c "\dt"
```

---

## Troubleshooting

### Контейнер уже существует

```bash
# Ошибка: container name is already in use
docker rm manga-crawler-db
docker-compose up -d
```

### Orphan контейнеры

```bash
# Ошибка: Found orphan containers
docker-compose up -d --remove-orphans
```

### Порт занят

```bash
# Ошибка: port is already allocated
# Найти процесс
netstat -ano | findstr :5432

# Или изменить порт в docker-compose.yml
```

### Приложение не видит БД

```bash
# Проверить что БД запущена
docker-compose ps

# Проверить подключение
docker exec -it manga-crawler-db pg_isready -U mangauser -d mangadb

# Посмотреть логи БД
docker-compose logs manga-crawler-db
```

### Миграции не применились

```bash
# Проверить таблицу миграций
docker exec -it manga-crawler-db psql -U mangauser -d mangadb -c "SELECT * FROM goose_db_version;"

# Перезапустить приложение
docker-compose restart manga-crawler-backend
```
