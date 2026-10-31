# Правила для AI-агентов (manga-crawler miniapp)

Применяйте при работе в каталоге `miniapp/`. Дополняют общие правила
репозитория; при расхождении приоритет у явного указания человека.
План и статус работ — [readme.md](readme.md).

## Стек и отличия от admin/

Vite + React 19 + TS strict + Tailwind v4 + shadcn + TanStack Query +
i18next (RU + EN) + Biome + `@telegram-apps/sdk-react` (v3). В отличие от
админки: **без** TanStack Router/Table и react-hook-form —
hash-роутинг руками ([src/lib/router.ts](src/lib/router.ts)), форм нет.
Пакетный менеджер — **pnpm**.

## Переиспользование UI — вендоринг, не импорты

- Кросс-импортов `miniapp → admin` нет: два независимых Vite-проекта со
  своими деплой-циклами.
- shadcn-примитивы и токены темы «Graphite» **копируются** из `admin/src`
  в `miniapp/src/components/ui/` и `src/styles/globals.css` и живут
  независимо (штатная модель shadcn). При изменении темы/примитивов в
  admin — синхронизация вручную.
- В `globals.css` только семантические классы (`bg-background`,
  `text-muted-foreground`); сырые цвета запрещены.

## Компоненты и слои

Нейминг — как в `admin/AGENTS.md`: компоненты с версткой — PascalCase
(`Button.tsx`, `BottomNav.tsx`, `Search.tsx`), файлы без верстки —
camelCase без дефисов (`router.ts`, `telegram.ts`, `client.ts`), папки
фич — PascalCase с корневым файлом и `index.ts` (default-экспорт), импорт
фичи снаружи — только через папку.

- `src/components/` — только разметка. Логика — в `src/lib/` (роутер,
  telegram-bootstrap, утилиты) или в `src/api/`.
- `src/api/` — HTTP-клиент miniapi ([client.ts](src/api/client.ts)) и
  query-клиент. Все запросы — через `api()` из client.ts (он ставит
  заголовок `X-Telegram-Init-Data`); прямой `fetch` вне client.ts
  запрещён. Ручные типы API-ответов появятся вместе со спекой
  `backend/docs/openapi-miniapp.yaml` (этап 2) и генерируются из неё.
- `src/features/<Фича>/` — экраны; серверные запросы фичи — в `queries.ts`
  рядом с компонентом (этап 3).
- `src/hooks/` — общие хуки (`useBackButton`), файл camelCase с `use`.

## Telegram SDK

- Инициализация — только через [src/lib/telegram.ts](src/lib/telegram.ts)
  (`bootstrapTelegram()` из `main.tsx`): mock-окружение вне Telegram в
  dev, `init()`, восстановление `initData`, подключение CSS-переменных
  themeParams/viewport, монтирование BackButton. Вызовы `init`/`mount*`
  в компонентах запрещены.
- SDK v3 экспортирует сигналы и SafeWrapped-функции: перед вызовом
  mount-функций проверяйте `fn.isAvailable()`, опциональные вызовы — в
  try/catch (клиенты без поддержки компонента не должны ронять экран).
- Данные из initData для запросов — через клиент API; компоненты читают
  пользователя через `initData.user()`.
- `bootstrapTelegram` идемпотентен — StrictMode вызывает его дважды.

## Роутинг

- Только hash-роутинг (`#/search`, `#/subscriptions`, `#/help`,
  `#/manga/<id>`) — GitHub Pages не делает rewrite на index.html.
  Парсинг/навигация — `parseHash`/`navigate` из `src/lib/router.ts`,
  хук `useRoute` в компонентах; ручные `location.hash =` в компонентах
  запрещены.
- Vite `base: '/manga-crawler/'` — не менять без изменения деплоя.

## i18n

- Весь видимый текст — через `t()`; RU-словарь ([ru.json](src/i18n/ru.json))
  базовый, ключи добавляются в RU и EN одним изменением. Типизация ключей
  выводится из ru.json (см. `i18n.d.ts`), опечатка ловится на компиляции.
- Локаль: localStorage (`miniapp-locale`), иначе — язык Telegram-юзера
  (см. `applyInitialLocale` в telegram.ts).

## Env

- `VITE_API_BASE_URL` — полный URL miniapi на проде; в dev не задан,
  работает Vite-прокси `/mini-api → localhost:8082` (порт dev-сервера —
  5174, чтобы не пересекаться с админкой).

## Проверки перед сдачей

- `npx tsc --noEmit`, `npx @biomejs/biome check .`, `npx vite build` —
  все три зелёные; ключи i18n в RU и EN синхронны.
