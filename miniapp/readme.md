# Telegram Mini App — план работ

Статус: **этапы 1 (каркас), 2 (cmd/miniapi), 3 (экраны MVP) и 4 (интеграция
с ботом) выполнены**, этап 5 (деплой GitHub Pages) — workflow готов и
проверен локальной симуляцией прод-сборки; сам деплой — за коротким списком
ручных шагов (выставить miniapi наружу, две настройки в GitHub, merge в
master). Приёмки: этап 3 — полный сценарий в dev-режиме против живого
miniapi; этап 4 — go build/vet/test + tsc/biome/build, deep link
`?startapp=<id>` в dev-браузере открывает карточку, нечисловой параметр
игнорируется (приёмка на устройстве закрывается вместе с этапом 5).
Для dev-сценария — `npm run dev:auth` (валидный квиток initData, см.
miniapp/AGENTS.md); deep link в dev-браузере имитируется адресом
`http://localhost:5174/manga-crawler/?startapp=<id>`. Далее — этап 6
(сдача) и 7 (сайт с входом через Telegram). Стек зафиксирован — как
в `admin/`
(Vite + React 19 + TS strict + Tailwind v4 + shadcn + TanStack Query +
i18next, без TanStack Router/Table/Hook Form — hash-роутинг, форм нет).

## Роль Mini App в проекте

Mini App — **основной интерфейс пользователя**: поиск манги, управление
подписками, карточка манги. Команды бота (`/start`, `/find`, `/list`,
`/help`) сохраняются как дублирующий функционал — удобны для тестирования и
как запасной канал. Уведомления о новых главах по-прежнему рассылает бот.

## Зафиксированные решения

| Вопрос               | Решение                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Чтение глав          | **Не входит** в MVP и в ближайшие этапы (чужие сайты, анти-бот, трафик)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| Подход               | Поэтапно: сначала каркас, затем экраны                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| Бэкенд               | **Раздельные API**: отдельный процесс `cmd/miniapi` (:8082, `/mini-api`) — своя модель аутентификации (`initData` вместо admin-cookie) и контур безопасности. Обоснование: `miniapi` публичный (GitHub Pages ходит на него из браузера с CORS), тогда как `adminapi` остаётся приватным — фронт админки работает только локально, иногда через VPN, наружу не выставляется. Публичная экспозиция `miniapi` не должна открывать админский контур — разные процессы делают это невозможным по построению                                                                                     |
| Хостинг              | **Раздельный для фронтов.** GitHub Pages **этого же репозитория** отведена только miniapp: URL `SemenovDmitry.github.io/manga-crawler/`, Vite `base: '/manga-crawler/'`, CORS-origin `https://SemenovDmitry.github.io`. У Pages один сайт на репозиторий (корень фиксирован именем репо, workflow публикует `dist/` в корень артефакта — подпуть `/miniapp` в URL не нужен), поэтому `admin/` сюда не деплоится: на будущее у админки — **отдельный репозиторий со своей Pages** (напр. `SemenovDmitry.github.io/<admin-repo>/`) или другой хостинг; на одну Pages два фронта не совмещать |
| Языки                | RU + EN (i18next-подход, как в admin)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| Функциональный объём | Дублирование функционала бота: поиск (`/find`), мои подписки (`/list`), карточка манги, подписка/отписка, «кто я» (`/start`), справка (`/help`)                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| Стек фронтенда       | **Как в `admin/`**: Vite + React 19 + TS strict + Tailwind v4 + shadcn (компоненты и тема «Graphite» переиспользуются) + TanStack Query + i18next + Biome. Отличия от админки: без TanStack Router/Table/Hook Form — hash-роутинг руками, форм нет; Telegram SDK (`@telegram-apps/sdk-react`); lighter-бандл за счёт малого набора экранов                                                                                                                                                                                                                                                 |
| Переиспользование UI | **Вендоринг, не импорты**: кросс-импортов `miniapp → admin` нет (два независимых Vite-проекта, свои деплой-циклы — сцеплять их нельзя). shadcn-компоненты и Tailwind-токены темы «Graphite» **копируются** из `admin/` в `miniapp/src/components/ui/` — это штатная модель shadcn («скопируй исходник себе»); далее код живёт независимо. Осознанная цена — ручная синхронизация правок; при её удорожании — опция будущего: workspace-пакет `packages/ui` (npm workspaces)                                                                                                                |

| Bot token | Продовый токен бота используется и для разработки (он же тестовый); `TELEGRAM_BOT_TOKEN` уже в `.env` — отдельный тестовый бот не заводится |

## Осталось решить

Открытых вопросов нет. Порог свежести `initData` фиксируется значением по
умолчанию 24 ч (`MINIAPI_INITDATA_MAX_AGE`), может быть изменён без
пересмотра плана.

## Архитектура

```
Telegram WebView ── HTTPS ──> GitHub Pages (статика miniapp/)
        │                            │
        │ initData в заголовке       │ REST + CORS (origin github.io)
        ▼                            ▼
   t.me/bot/app  ────────────> cmd/miniapi (:8082, /mini-api)
                                      │
                                      ▼
                              Postgres (общая БД)
```

- **Аутентификация:** фронт берёт `initData` из Telegram SDK и передаёт в
  каждом запросе (заголовок `X-Telegram-Init-Data`). `miniapi` проверяет
  HMAC-SHA256 (секрет = `HMAC_SHA256(bot_token, "WebAppData")`), свежесть
  `auth_date` (порог ~24 ч), находит/создаёт пользователя в `chat_users`.
  Без сессий и cookie — проверка на каждый запрос, stateless.
- **CORS:** `miniapi` разрешает origin `https://SemenovDmitry.github.io`; cookies
  нет, поэтому SameSite-проблем не существует.
- **Роутинг:** hash-роутинг (`/#/manga/123`) — GitHub Pages не умеет
  rewrite на `index.html`.
- **Base path:** деплой в подпуть репозитория (`/manga-crawler/`) —
  настраивается `base` в Vite.
- **API URL:** в бандл через `VITE_API_BASE_URL` на этапе сборки.
- **Env `miniapi`:** `MINIAPI_LISTEN` (напр. `:8082`),
  `MINIAPI_ALLOWED_ORIGIN`, `TELEGRAM_BOT_TOKEN` (уже есть),
  `MINIAPI_INITDATA_MAX_AGE`. В bot добавляется `MINIAPP_URL` для Menu
  Button.

## Этапы работ

### Этап 1 — каркас miniapp (фронтенд)

- Vite-проект в `miniapp/` на зафиксированном стеке (см. таблицу решений),
  TS strict, Tailwind v4.
- Подключение Telegram SDK: тема (`themeParams`), `initData`,
  `BackButton`, viewport.
- Dev-режим: mock `initData` (вне Telegram данных нет), Vite-прокси
  `/mini-api → localhost:8082` (как в admin).
- Каркас i18n: `ru.json` + `en.json`, ключи всегда парой.
- Заглушки экранов: Поиск, Подписки, Карточка, Справка.
- `miniapp/AGENTS.md`: правила нейминга, слоёв, i18n, проверки.

Приёмка: `npm run dev` показывает заглушки в браузере с mock-данными;
`npx tsc --noEmit`, линтер, `vite build` — зелёные.

### Этап 2 — cmd/miniapi (бэкенд)

- Спека `backend/docs/openapi-miniapp.yaml` (OpenAPI 3.0.3), эндпоинты:
  - `GET /mini-api/me` — текущий пользователь (из initData);
  - `GET /mini-api/manga/search?q=` — поиск по названию;
  - `GET /mini-api/manga/{id}` — карточка манги + подписан ли я;
  - `GET /mini-api/subscriptions` — мои подписки (аналог `/list`);
  - `PUT /mini-api/subscriptions/{mangaId}` / `DELETE` — подписка/отписка;
  - `GET /mini-api/help` — статический текст справки (или без API, из i18n).
- oapi-codegen по образцу `internal/adminapi` (генерат не править руками).
- Middleware валидации `initData` + привязка к `chat_users`.
- Переиспользование `internal/storage`: `Subscribe`, `Unsubscribe`,
  `UserSubscriptions` уже есть; при необходимости добавить
  `SearchManga(query)` (поиск для `/find` сейчас в `internal/telegram` —
  вынести общий запрос в storage). Миграций, по-видимому, не требуется.
- Запись в корневом `docker compose` и `backend/infrastructure.md`.

Приёмка: curl с валидным/невалидным `initData` — 200/401; спека и генерат
в коммите вместе.

### Этап 3 — экраны MVP

- Поиск: поле + debounce-запрос, список карточек-строк.
- Подписки: список с обложкой/статусом, свайп или кнопка отписки,
  optimistic update через TanStack Query.
- Карточка манги: обложка, статус публикации, источник, главы,
  кнопка «Подписаться/Отписаться».
- Справка: статичный экран RU/EN.
- Skeleton-загрузка, пустые состояния, ошибки — в теме Telegram
  (`themeParams`, поддержка светлой/тёмной).

Приёмка: полный сценарий в dev-режиме с mock-данными; `tsc`/линтер/build
зелёные; ключи i18n RU+EN синхронны.

### Этап 4 — интеграция с ботом (выполнен)

- Menu Button бота → URL мини-аппа (`MINIAPP_URL`): при старте бот вызывает
  `setChatMenuButton` (web_app-кнопка); переменная не задана — меню команд
  не трогается.
- `web_app`-кнопка «Открыть в приложении» завершает клавиатуру результатов
  `/find` (тоже только при заданном `MINIAPP_URL`).
- Deep links `?startapp=<manga_id>` → мини-апп открывает карточку:
  Telegram передаёт параметр в launch-параметре `tgWebAppStartParam`
  (попадает в query загруженной страницы), роутер (`applyStartDeepLink`
  в `src/lib/router.ts`) применяет его до первого рендера; нечисловой
  параметр игнорируется. В dev-браузере имитация — просто `?startapp=<id>`
  в адресе. Тесты: `TestFindResultsKeyboardWithMiniApp`,
  `TestMiniAppButtonRow` (`internal/telegram`).
- Тексты бота упоминают мини-апп (`/help`).

Приёмка: go build/vet/test + tsc/biome/build зелёные; deep link в
dev-браузере открывает карточку (id 45728), пустой/нечисловой параметр —
стартовый экран. Приёмка на устройстве (кнопка меню, web_app-кнопка в
чате) — после этапа 5: web_app требует HTTPS, без продового URL
`MINIAPP_URL` остаётся пустым.

### Этап 5 — деплой GitHub Pages (workflow готов, деплой — ручные шаги)

- GitHub Actions: `.github/workflows/miniapp-pages.yml` — триггеры: push в
  `master` при изменениях `miniapp/**` (или самого workflow) + ручной
  запуск; шаги: `pnpm install --frozen-lockfile` → `pnpm build`
  (tsc + vite) → публикация `miniapp/dist` в корень артефакта Pages
  (`actions/deploy-pages`). CI использует pnpm — как в проекте
  (pnpm-lock.yaml; npm в miniapp не используется, см. AGENTS.md).
- `VITE_API_BASE_URL` — из переменной репозитория `MINIAPI_PUBLIC_URL`,
  вшивается в бандл при сборке; переменная не задана — workflow падает
  с внятной ошибкой. Сменили адрес miniapi — перезапустить workflow.
- Vite `base: '/manga-crawler/'` — уже настроен (этап 1); локальная
  симуляция прод-сборки зелёная: `VITE_API_BASE_URL=... pnpm build` →
  ассеты под `/manga-crawler/assets/`, URL вшит в бандл, `vite preview`
  отдаёт `/manga-crawler/` и ассеты.
- CORS: `MINIAPI_ALLOWED_ORIGIN=https://SemenovDmitry.github.io` —
  прод-значение задокументировано; требования публичной экспозиции
  miniapi (HTTPS, наружу только miniapi, пример реверс-прокси) —
  в backend/infrastructure.md («Публичная экспозиция»).

Ручные шаги деплоя (в порядке):

1. Выставить miniapi наружу по HTTPS: любой реверс-прокси/туннель с
   правилом `${MINIAPI_PUBLIC_URL}/mini-api/* → miniapi:8082`; в .env
   прод-хоста — `MINIAPI_ALLOWED_ORIGIN=https://SemenovDmitry.github.io`.
2. GitHub: Settings → Secrets and variables → Actions → Variables →
   переменная `MINIAPI_PUBLIC_URL` (полный адрес miniapi, без `/mini-api`
   и без слэша на конце).
3. GitHub: Settings → Pages → Source: «GitHub Actions».
4. Влить `develop` в `master` (или запустить workflow вручную) — Pages
   соберёт и опубликует `https://SemenovDmitry.github.io/manga-crawler/`.
5. Menu Button: отдельный шаг в BotFather не нужен — бот сам ставит кнопку
   через `setChatMenuButton`, если в .env прод-хоста задан `MINIAPP_URL`
   (этап 4); выставить и перезапустить бота.
6. Проверка с телефона: приложение открывается с продового URL, поиск и
   подписки работают против продового miniapi; deep link
   `t.me/<бот>?startapp=<id>` открывает карточку — этим закрывается и
   отложенная приёмка этапа 4 (кнопка меню, web_app-кнопка в `/find`).

Приёмка: приложение открывается с продового URL на устройстве, поиск и
подписки работают против продового `miniapi` (выполняется после ручных
шагов).

### Этап 6 — сдача

- Обновить корневой `AGENTS.md` (строка miniapp) и `backend/README.md`.
- Ручной чек-лист: RU/EN, тёмная/светлая тема, iOS + Android + Desktop
  Telegram, оффлайн-ошибки, истёкший `initData` (переоткрытие).

### Этап 7 — работа через сайт с входом через Telegram (Login Widget)

Приложение уже открывается в любом браузере (SPA на Pages); вне Telegram
не работает только аутентификация (initData не существует → 401).
Добавить официальный Telegram Login Widget — сайт показывает кнопку
«Войти через Telegram», виджет отдаёт JSON пользователя + `auth_date` +
`hash`.

- **Фронтенд**: экран входа (показывается вне Telegram), подключение
  скрипта виджета; `client.ts` выбирает квиток: в Telegram — `initData`,
  на сайте — payload виджета (в памяти сессии).
- **miniapi**: вторая функция валидации рядом с `validateInitData` —
  **другая математика**: `hash = HMAC_SHA256(data-check-string,
  key = SHA256(bot_token))` (у initData ключ = `HMAC(bot_token,
  "WebAppData")`). Middleware принимает оба заголовка; upsert в
  `chat_users` и порог `MINIAPI_INITDATA_MAX_AGE` — без изменений;
  CORS не трогается (origin тот же).
- **Спека**: второй securityScheme в `openapi-miniapp.yaml`, регенерация
  типов и генерата.
- **Тесты**: юнит-тесты на обе функции валидации (закрывает известный
  пробел; смотри ловушку в истории: порядок аргументов HMAC проверять по
  докам Telegram, не по «работает же»).
- **BotFather** (ручной шаг): `/setdomain` для бота с доменом
  `SemenovDmitry.github.io` — без этого виджет на домене откажется
  работать.

Приёмка: в обычном браузере (не Telegram) экран входа → вход через
виджет → поиск/подписки работают тем же пользователем `chat_users`.

Альтернатива (отклонена на данный момент): QR-бридж через бота как
WhatsApp Web — требует сессий в БД (миграция, которой план осознанно
избегает) и эндпоинта поллинга; вернуться, если UX oauth-попапа не
устроит.

## Что не входит (осознанно)

- Чтение глав и любые картинки-страницы.
- Публичные страницы с OG-разметкой.
- Каталог целиком/бесконечная лента (поиск + подписки покрывают сценарий).
- Push-уведомления из мини-аппа (уведомляет бот, как и сейчас).
