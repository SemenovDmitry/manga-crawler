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
| miniapi | cmd/miniapi | 8082 | HTTP API мини-аппа Telegram (см. ниже) |

Миграции зашиты в бинарники (goose) и применяются каждым процессом при старте
под advisory-блокировкой — порядок запуска контейнеров не важен.

---

## Admin API — запуск

Контракт: `docs/openapi-admin.yaml` (OpenAPI 3.0.3) — единый источник правды.
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

    curl -i http://localhost:8081/admin-api/auth/login \
      -H "Content-Type: application/json" \
      -d '{"username":"admin","password":"ваш-пароль"}'
    # 200 + Set-Cookie admin_access/admin_refresh (HttpOnly) — API работает

---

## Admin API — запросы

Аутентификация — через HttpOnly-cookie: `login` ставит `admin_access` и
`admin_refresh` (SameSite=Strict, Path=/admin-api). Дальнейшие запросы cookie
подставляет сам клиент (браузер, Postman cookie-jar, `curl -b cookies.txt`).

### Порядок работы

1. `POST /admin-api/auth/login` — сервер ставит пару cookie
   (access живёт 15 минут, refresh — 7 дней; `ADMIN_COOKIE_SECURE=true`
   включает флаг Secure — только за TLS).
2. Дальнейшие запросы — с cookie (клиент подставляет автоматически).
3. Access истёк (401) → `POST /admin-api/auth/refresh` — читает refresh-cookie,
   выдаёт новую пару (снова в cookie). Refresh одноразовый: повторное
   использование старого → 401, обе пары считаются скомпрометированными
   (нужно перелогиниться).
4. `POST /admin-api/auth/logout` — отозвать refresh-сессию и погасить cookie.

### Примеры (bash, cookie-jar)

    # логин (cookie сохраняются в файл)
    curl -s -c cookies.txt http://localhost:8081/admin-api/auth/login \
      -H "Content-Type: application/json" \
      -d '{"username":"admin","password":"ваш-пароль"}'

    # список источников
    curl -s -b cookies.txt http://localhost:8081/admin-api/sources

    # выключить источник
    curl -s -b cookies.txt -X PATCH http://localhost:8081/admin-api/sources/1/enabled \
      -H "Content-Type: application/json" \
      -d '{"enabled":false}'

    # создать мангу
    curl -s -b cookies.txt -X POST http://localhost:8081/admin-api/manga \
      -H "Content-Type: application/json" \
      -d '{"source_id":1,"url":"test_1","title":"Тест"}'

    # частичный PATCH — только переданные поля меняются
    curl -s -b cookies.txt -X PATCH http://localhost:8081/admin-api/manga/42 \
      -H "Content-Type: application/json" \
      -d '{"title":"Новое название"}'

    # удалить — в ответе удалённая запись (последнее состояние перед удалением)
    curl -s -b cookies.txt -X DELETE http://localhost:8081/admin-api/manga/42

    # обновить пару токенов (ротация refresh-cookie)
    curl -s -b cookies.txt -c cookies.txt http://localhost:8081/admin-api/auth/refresh

    # выход
    curl -s -b cookies.txt -X POST http://localhost:8081/admin-api/auth/logout

Коды ответов: 200 — успех (DELETE тоже — с телом удалённой записи),
201 — создано, 204 — без тела (logout),
400 — неверные данные (тело/параметры, включая enum `status` и `limit`),
401 — нет/просрочен access или плохой refresh, 404 — не найдено,
409 — дубликат манги (тот же source_id + url),
500 — внутренняя ошибка (детали только в логах сервера).

### Postman

Готовая коллекция с тестами: `postman/admin-api.postman_collection.json`.

1. Postman → Import → выбрать файл коллекции.
2. В переменных коллекции задать `username` и `password` (ваши ADMIN_USERNAME
   и пароль; `baseUrl` по умолчанию http://localhost:8081).
3. Запустить папку **Auth** — `login` поставит HttpOnly-cookie, Postman
   сохранит их в cookie-jar, остальные запросы отправят их автоматически.
4. Папка **Manga CRUD** — полный цикл create → get → update → delete → 404;
   запускать через **Collection Runner** (порядок важен: create сохраняет
   `mangaId`, update проверяет частичность PATCH).
5. Папка **Errors (негативные)** — 401 без cookie, 401 с неверным паролем,
   400 на enum `status` и `limit`, 404 на несуществующую мангу.

Прогон всех тестов: Runner → выбрать коллекцию → Run. Все запросы содержат
тесты (проверки статусов, структуры ответов, ротации refresh, частичности
PATCH) — результат во вкладке Test Results.

Альтернатива: импортировать `docs/openapi-admin.yaml` как OpenAPI — Postman
соберёт коллекцию без тестов, но с актуальными схемами. Из той же схемы
позже генерируется TS-клиент для admin-фронтенда (Orval).

---

## Mini API — запуск (cmd/miniapi)

Публичный API мини-аппа Telegram: поиск манги, подписки. В отличие от
adminapi, выставляется наружу (мини-апп на GitHub Pages ходит на него из
браузера), поэтому вынесен в отдельный процесс — публичная экспозиция
не открывает админский контур.

Контракт: `docs/openapi-miniapp.yaml` (OpenAPI 3.0.3) — единый источник
правды. Генерат: `internal/miniapi/oapi/api.gen.go` (не править руками):

    go generate ./internal/miniapi/...

Аутентификация — stateless: каждый запрос несёт заголовок
`X-Telegram-Init-Data` с сырым initData из Telegram SDK. Сервер проверяет
HMAC-SHA256 (секрет — `HMAC_SHA256(bot_token, "WebAppData")`), свежесть
`auth_date` (`MINIAPI_INITDATA_MAX_AGE`, по умолчанию 24 ч) и создаёт/
обновляет пользователя в `chat_users`. Cookie и сессий нет; CORS-origin —
`MINIAPI_ALLOWED_ORIGIN` (прод: `https://SemenovDmitry.github.io`; в dev
пусто — фронт ходит через Vite-прокси).

### Переменные (.env)

Обязательна только `TELEGRAM_BOT_TOKEN` (уже есть — та же, что у бота);
остальные опциональны (см. .env.example):

    MINIAPI_LISTEN=:8082
    MINIAPI_ALLOWED_ORIGIN=
    # MINIAPI_INITDATA_MAX_AGE=24h

### Запуск

Docker:

    docker compose up -d --build miniapi
    # miniapi: http://localhost:8082/mini-api/...
    docker compose logs -f miniapi

Локально (нужен доступный Postgres из .env):

    cd backend
    go run ./cmd/miniapi

### Проверка (curl)

Квиток initData подписывается бот-токеном — для ручного теста его можно
вычислить (bash):

    TOKEN="ваш TELEGRAM_BOT_TOKEN"
    USER='{"id":777000001,"first_name":"Dev","username":"dev_user","language_code":"ru"}'
    AUTH_DATE=$(date +%s)
    # data-check-string: все поля квитка, кроме hash, по алфавиту, разделитель \n
    DCS=$(printf 'auth_date=%s\nuser=%s' "$AUTH_DATE" "$USER")
    # секрет: HMAC_SHA256(данные=bot_token, ключ="WebAppData")
    SECRET=$(printf '%s' "$TOKEN" | openssl dgst -sha256 -mac HMAC -macopt "key:WebAppData" -binary | xxd -p -c 64)
    HASH=$(printf '%s' "$DCS" | openssl dgst -sha256 -mac HMAC -macopt "hexkey:$SECRET" | awk '{print $2}')
    INITDATA="auth_date=$AUTH_DATE&user=$(printf %s "$USER" | jq -sRr @uri)&hash=$HASH"

    # без заголовка → 401
    curl -i http://localhost:8082/mini-api/me
    # с битым hash → 401
    curl -i http://localhost:8082/mini-api/me -H "X-Telegram-Init-Data: auth_date=$AUTH_DATE&user=...&hash=bad"
    # с валидным квитком → 200 и данные пользователя
    curl -s http://localhost:8082/mini-api/me -H "X-Telegram-Init-Data: $INITDATA"

    # поиск, карточка, подписки
    curl -s "http://localhost:8082/mini-api/manga/search?q=<название>" -H "X-Telegram-Init-Data: $INITDATA"
    curl -s http://localhost:8082/mini-api/manga/42 -H "X-Telegram-Init-Data: $INITDATA"
    curl -s -X PUT http://localhost:8082/mini-api/subscriptions/42 -H "X-Telegram-Init-Data: $INITDATA"
    curl -s http://localhost:8082/mini-api/subscriptions -H "X-Telegram-Init-Data: $INITDATA"
    curl -s -X DELETE http://localhost:8082/mini-api/subscriptions/42 -H "X-Telegram-Init-Data: $INITDATA"

Коды ответов: 200 — успех (PUT/DELETE отдают состояние подписки),
400 — неверные параметры (битый mangaId, limit вне диапазона),
401 — нет/битый/просроченный initData или неверная подпись,
404 — манга не найдена, 500 — внутренняя ошибка (детали в логах).
