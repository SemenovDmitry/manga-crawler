# Infrastructure — manga-crawler-backend

Проект — Telegram-бот на чистом Go (без веб-фреймворков и БД). Единственная внешняя
зависимость — `github.com/joho/godotenv` (чтение `.env`).

Как запустить
  в корне backend/:
  docker compose up -d --build

go mod tidy - Вызывать после изменения зависимойстей перед билдом и коммитом
  синхронизирует go.mod и go.sum с реальным импортами кода: добавляет недостающие зависимости, убирает неиспользуемые, досчитывает хэши в go.sum.

## Состав

| Сервис | Процесс | Порт | Назначение |
|---|---|---|---|
| db | postgres:16-alpine | 5432 | Единая БД, том db-data |
| bot | cmd/bot | 8080 | Telegram-бот, рассылка уведомлений |
| crawler | cmd/crawler | — | Обход источников |
| adminapi | cmd/adminapi | 8081 | HTTP API админ-панели (см. ниже) |

Миграции зашиты в бинарники (goose) и применяются каждым процессом при старте
под advisory-блокировкой — порядок запуска контейнеров не важен.

---

## Admin API — запуск

Контракт: `docs/openapi.yaml` (OpenAPI 3.0.3) — единый источник правды.
Из него oapi-codegen генерирует Go-типы и интерфейс хендлеров:
`internal/adminapi/oapi/api.gen.go` (генерируется, не править руками).
Регенерация после изменения схемы:

    go generate ./internal/adminapi/...

(конфиг: internal/adminapi/oapi/cfg.yaml; версия инструмента зафиксирована
в go.mod директивой `tool`). Хендлеры internal/adminapi обязаны реализовывать
`oapi.ServerInterface` — проверка на этапе компиляции. Тесты пакета
(`go test ./internal/adminapi/...`) сверяют роутер со схемой и проверяют
auth-флоу; Postman-коллекция — в `postman/` (см. ниже).

### 1. Настроить .env (backend/.env)

Обязательные для adminapi переменные (см. .env.example):

    ADMIN_LISTEN_ADDR=:8081
    ADMIN_USERNAME=admin
    ADMIN_PASSWORD_HASH=...   # bcrypt-хэш, генерируется командой ниже
    JWT_SECRET=...            # openssl rand -hex 32

Хэш пароля:

    cd backend
    go run ./cmd/hashpassword
    # → вставьте выведенный хэш в ADMIN_PASSWORD_HASH

Секрет JWT (минимум 32 байта, иначе сервер откажется стартовать):

    openssl rand -hex 32

### 2. Запуск

Docker (вместе с ботом и краулером; сервис adminapi уже в docker-compose.yml):

    docker compose up -d --build
    # adminapi: http://localhost:8081/admin-api/...
    docker compose logs -f adminapi   # логи

Локально (нужен доступный Postgres из .env):

    cd backend
    go run ./cmd/adminapi

Быстрая проверка живости:

    curl http://localhost:8081/admin-api/auth/login \
      -H "Content-Type: application/json" \
      -d '{"username":"admin","password":"ваш-пароль"}'
    # 200 + access_token/refresh_token — API работает

---

## Admin API — запросы

Все защищённые эндпоинты требуют заголовок:

    Authorization: Bearer <access_token>

### Порядок работы

1. `POST /admin-api/auth/login` — получить пару токенов
   (access живёт 15 минут, refresh — 7 дней).
2. Дальнейшие запросы — с Bearer access-токеном.
3. Access истёк (401) → `POST /admin-api/auth/refresh` с refresh-токеном —
   выдаётся новая пара. Refresh одноразовый: повторное использование старого
   → 401, обе пары считаются скомпрометированными (нужно перелогиниться).
4. `POST /admin-api/auth/logout` — отозвать refresh-сессию при выходе.

### Примеры (bash)

    # логин
    TOKENS=$(curl -s http://localhost:8081/admin-api/auth/login \
      -H "Content-Type: application/json" \
      -d '{"username":"admin","password":"ваш-пароль"}')
    ACCESS=$(echo $TOKENS | jq -r .access_token)
    REFRESH=$(echo $TOKENS | jq -r .refresh_token)

    # список источников
    curl -s http://localhost:8081/admin-api/sources -H "Authorization: Bearer $ACCESS"

    # выключить источник
    curl -s -X PATCH http://localhost:8081/admin-api/sources/1/enabled \
      -H "Authorization: Bearer $ACCESS" -H "Content-Type: application/json" \
      -d '{"enabled":false}'

    # создать мангу
    curl -s -X POST http://localhost:8081/admin-api/manga \
      -H "Authorization: Bearer $ACCESS" -H "Content-Type: application/json" \
      -d '{"source_id":1,"url":"test_1","title":"Тест"}'

    # частичный PATCH — только переданные поля меняются
    curl -s -X PATCH http://localhost:8081/admin-api/manga/42 \
      -H "Authorization: Bearer $ACCESS" -H "Content-Type: application/json" \
      -d '{"title":"Новое название"}'

    # удалить — в ответе удалённая запись (последнее состояние перед удалением)
    curl -s -X DELETE http://localhost:8081/admin-api/manga/42 \
      -H "Authorization: Bearer $ACCESS"

    # обновить пару токенов
    curl -s http://localhost:8081/admin-api/auth/refresh \
      -H "Content-Type: application/json" -d "{\"refresh_token\":\"$REFRESH\"}"

    # выход
    curl -s -X POST http://localhost:8081/admin-api/auth/logout \
      -H "Authorization: Bearer $ACCESS" -H "Content-Type: application/json" \
      -d "{\"refresh_token\":\"$REFRESH\"}"

Коды ответов: 200 — успех (DELETE тоже — с телом удалённой записи),
201 — создано, 204 — без тела (logout),
400 — неверные данные (тело/параметры, включая enum `status` и `limit`),
401 — нет/просрочен access или плохой refresh, 404 — не найдено,
500 — внутренняя ошибка (детали только в логах сервера).

### Postman

Готовая коллекция с тестами: `postman/admin-api.postman_collection.json`.

1. Postman → Import → выбрать файл коллекции.
2. В переменных коллекции задать `username` и `password` (ваши ADMIN_USERNAME
   и пароль; `baseUrl` по умолчанию http://localhost:8081).
3. Запустить папку **Auth** — `login` сохранит `accessToken`/`refreshToken`
   в переменные, остальные запросы подставят их автоматически.
4. Папка **Manga CRUD** — полный цикл create → get → update → delete → 404;
   запускать через **Collection Runner** (порядок важен: create сохраняет
   `mangaId`, update проверяет частичность PATCH).
5. Папка **Errors (негативные)** — 401 без токена, 401 с неверным паролем,
   400 на enum `status` и `limit`, 404 на несуществующую мангу.

Прогон всех тестов: Runner → выбрать коллекцию → Run. Все запросы содержат
тесты (проверки статусов, структуры ответов, ротации refresh, частичности
PATCH) — результат во вкладке Test Results.

Альтернатива: импортировать `docs/openapi.yaml` как OpenAPI — Postman
соберёт коллекцию без тестов, но с актуальными схемами. Из той же схемы
позже генерируется TS-клиент для admin-фронтенда (Orval).
