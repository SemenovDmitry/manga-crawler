# manga-crawler — навигация по проекту

Монорепозиторий сервиса отслеживания обновлений манги: краулер обходит
сайты-источники, детектит новые главы, Telegram-бот рассылает уведомления
подписчикам, админ-панель управляет источниками и каталогом манги.

## Структура репозитория

| Каталог    | Проект                                             | Подробные правила и описание |
| ---------- | -------------------------------------------------- | ---------------------------- |
| `backend/` | Go-бэкенд: бот, краулер, admin API, БД Postgres    | [backend/AGENTS.md](backend/AGENTS.md), [backend/README.md](backend/README.md), [backend/infrastructure.md](backend/infrastructure.md) |
| `admin/`   | Веб-админ-панель (Vite + React 19 + TypeScript)    | [admin/AGENTS.md](admin/AGENTS.md), [admin/readme.md](admin/readme.md) |
| `miniapp/` | Telegram Mini App (заготовка)                      | [miniapp/readme.md](miniapp/readme.md) |

Правила для агентов каждого проекта лежат в его `AGENTS.md` — читайте файл
проекта **перед** началом работы в его каталоге; они дополняют эти правила,
при расхождении приоритет у более специфичного файла.

## backend/ — Go-монолит из трёх процессов

Модуль `github.com/SemenovDmitry/manga-crawler-backend`, Go 1.26, Postgres
(docker compose, `db` на :5432).

- **`cmd/bot`** — Telegram-бот: long polling, команды (`/start`, `/find`,
  `/list`, `/help`), рассылка уведомлений из очереди. Логика в
  `internal/telegram`.
- **`cmd/crawler`** — обход источников: полный обход по расписанию
  (`sources.next_full_crawl_at`, по умолчанию 2 недели) и инкрементальный
  каждые `CRAWL_INTERVAL` (20 минут). Драйверы движков сайтов —
  `internal/crawler/parsers/{grouple,mgeko}`; новый сайт на известном движке
  добавляется INSERT-ом в `sources`, новый движок — реализацией
  `SourceDriver` + case в `internal/crawler/registry.go`.
- **`cmd/adminapi`** — HTTP API админ-панели (:8081, префикс `/admin-api`).
  Контракт — `docs/openapi-admin.yaml` (OpenAPI 3.0.3), из него oapi-codegen
  генерирует `internal/adminapi/oapi/api.gen.go` (не править руками;
  регенерация — `go generate ./internal/adminapi/...`).
- **`internal/storage`** — доступ к Postgres, миграции goose в
  `internal/storage/migrations`, применяются процессами при старте под
  advisory-блокировкой. Снимок схемы — `docs/schema.sql` (производный
  артефакт, см. правило в `backend/AGENTS.md`).
- **Служебные**: `cmd/hashpassword` (bcrypt-хэш для `.env`),
  `cmd/schemadump` (снимок схемы), `cmd/probe` (разведка сайтов, запускается
  на машине пользователя — сайты блокируют датацентровые IP).

Ключевое правило (`backend/AGENTS.md`): любое изменение в
`internal/storage/migrations/` → сразу обновить `docs/schema.sql` командой
`go run ./cmd/schemadump` из `backend/` (нужен контейнер `db`), коммитить
вместе с миграцией, руками не редактировать.

## admin/ — веб-админ-панель

SPA над `cmd/adminapi`; контракт API — `backend/docs/openapi-admin.yaml`.

- **Стек**: Vite + React 19 + TypeScript strict, Tailwind CSS v4 +
  shadcn-компоненты (в репозитории, тёмная «Graphite»-тема), TanStack Router
  (файл-роутинг, гарды), TanStack Query, TanStack Table, react-hook-form +
  zod, react-i18next (RU базовый + EN), Biome.
- **API-клиент**: генерируется `npm run gen:api` из OpenAPI-спеки бэкенда в
  `src/api/schema.gen.ts` — ручные типы, дублирующие схему, запрещены.
  Аутентификация — HttpOnly-cookie, при 401 автоматический refresh с
  очередью запросов (middleware, код страниц своей логики refresh не пишет).
- **Структура**: `src/api` (клиент + auth), `src/features/{Sources,
  ChatUsers,Manga,...}` (фичи), `src/components/ui` (shadcn), `src/lib`
  (сторы и утилиты), `src/i18n/{ru,en}.json`, `src/routes` (тонкие
  файлы-роуты TanStack Router).
- **Маршруты**: `/login`, `/` → `/manga`, `/sources`, `/chat-users`,
  `/manga`, `/manga/new`, `/manga/$id`.
- **Запуск (dev)**: `npm run gen:api` затем `npm run dev` (:5173, Vite
  проксирует `/admin-api → localhost:8081`). Пакетный менеджер — npm
  (pnpm в этой среде не работает, см. `admin/readme.md`).
- **Проверки перед сдачей**: `npx tsc --noEmit`,
  `npx @biomejs/biome check .`, `npx vite build` — все зелёные; новые ключи
  i18n — в RU и EN одновременно.

Детальные правила нейминга, слоёв и данных — в `admin/AGENTS.md`; этапы и
статус разработки — в `admin/readme.md`.

## miniapp/ — Telegram Mini App

Заготовка (`App for telegram`), спецификация в `miniapp/readme.md`.

## Связи между проектами

- `admin/` зависит от `backend/docs/openapi-admin.yaml`: изменили спеку в бэкенде —
  перегенерируйте клиент админки (`npm run gen:api`).
- `adminapi` (бэкенд) и админ-панель (фронтенд) — единый контракт, cookie-auth
  с `Path=/admin-api`, dev-прокси Vite вместо CORS.
- Все процессы бэкенда работают с одной базой Postgres и применяют одни и те
  же миграции.
