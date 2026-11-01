/**
 * Bootstrap Telegram SDK: init, mock-режим вне Telegram, подключение
 * CSS-переменных темы и viewport.
 *
 * Порядок важен: mockTelegramEnv (если он нужен) вызывается строго ДО init —
 * SDK должен увидеть launch-параметры при инициализации.
 */
import {
  bindMiniAppCssVars,
  bindThemeParamsCssVars,
  bindViewportCssVars,
  expandViewport,
  init,
  initData,
  isTMA,
  mockTelegramEnv,
  mountBackButton,
  mountMiniAppSync,
  mountThemeParamsSync,
  mountViewport,
} from "@telegram-apps/sdk-react"

import { changeLocale, defaultLocale, LOCALE_STORAGE_KEY, locales } from "@/i18n"

/** Dev-квиток initData. Приоритет — валидный квиток, подписанный бот-токеном
 * (scripts/dev-initdata.mjs → .env.local): запросы через Vite-прокси проходят
 * проверку подписи в miniapi. Фолбэк — структурно валидный квиток с
 * фиктивным hash: годится для экранов, но API отвечает 401. */
function mockInitData(): string {
  const signed = import.meta.env.VITE_DEV_INIT_DATA
  if (signed) {
    return signed
  }
  const user = {
    id: 777_000_001,
    first_name: "Dev",
    last_name: "User",
    username: "dev_user",
    language_code: "ru",
  }
  return new URLSearchParams({
    auth_date: String(Math.floor(Date.now() / 1000)),
    hash: "mock-hash",
    signature: "mock-signature",
    user: JSON.stringify(user),
  }).toString()
}

/** Тёмная тема Telegram Desktop — подставляется как tgWebAppThemeParams,
 * чтобы mountThemeParams в dev-браузере видел реалистичные цвета. */
const MOCK_THEME_PARAMS = {
  bg_color: "#212121",
  button_color: "#5288c1",
  button_text_color: "#ffffff",
  hint_color: "#aaaaaa",
  link_color: "#8774e1",
  secondary_bg_color: "#181818",
  header_bg_color: "#212121",
  accent_text_color: "#6ab3f3",
  section_bg_color: "#181818",
  section_header_text_color: "#8774e1",
  section_separator_color: "#2b2b2b",
  subtitle_text_color: "#aaaaaa",
  text_color: "#ffffff",
  bottom_bar_bg_color: "#181818",
  destructive_text_color: "#ec3942",
}

/** Локаль Telegram-пользователя — только при первом запуске; явный выбор
 * пользователя в приложении имеет приоритет (см. LocaleSwitch). */
function applyInitialLocale(languageCode: string | undefined) {
  if (localStorage.getItem(LOCALE_STORAGE_KEY)) {
    return
  }
  const locale = locales.find((l) => l === languageCode)
  if (locale && locale !== defaultLocale) {
    changeLocale(locale)
  }
}

let bootstrapped = false

/**
 * Инициализация SDK. Вне Telegram включает mock-окружение с dev-данными
 * (только в dev-сборке). Все монтирования — опциональны: на платформах без
 * поддержки компонента они тихо пропускаются.
 *
 * @returns true, если работаем в настоящем Telegram-клиенте.
 */
export function bootstrapTelegram(): boolean {
  if (bootstrapped) {
    return isTMA()
  }
  bootstrapped = true

  if (!isTMA()) {
    if (!import.meta.env.DEV) {
      // Прод вне Telegram (прямой заход на Pages) — работаем без SDK.
      return false
    }
    // Launch-параметры передаём строкой — как их присылает настоящий клиент:
    // tgWebAppThemeParams обязателен и парсится из JSON с snake_case-ключами.
    mockTelegramEnv({
      launchParams: new URLSearchParams({
        tgWebAppData: mockInitData(),
        tgWebAppVersion: "7.10",
        tgWebAppPlatform: "tdesktop",
        tgWebAppThemeParams: JSON.stringify(MOCK_THEME_PARAMS),
      }).toString(),
    })
  }

  // init() завершается постом iframe_ready; в dev-браузере верхнего уровня
  // у postEvent нет моста, и этот пост бросает UnknownEnvError — хотя к тому
  // моменту SDK уже инициализирован. В настоящем Telegram мост есть, броска
  // не будет, поэтому глушим только здесь.
  try {
    init()
  } catch {
    // нет моста для iframe_ready — конфигурация SDK уже применена
  }

  // Восстанавливаем initData из launch-параметров: из него берём и локаль
  // пользователя, и квиток для заголовка X-Telegram-Init-Data.
  // restore — не SafeWrapped-функция (в отличие от mount*), isAvailable нет.
  initData.restore()
  applyInitialLocale(initData.user()?.language_code)

  // Тема приходит в launch-параметрах — монтируем синхронно: у bind*
  // isAvailable() требует смонтированный компонент, и после async-mount
  // проверка не успевает (см. примечание ниже про mount*.then).
  if (mountThemeParamsSync.isAvailable()) {
    mountThemeParamsSync()
    if (bindThemeParamsCssVars.isAvailable()) {
      bindThemeParamsCssVars()
    }
  }
  // Асинхронное монтирование viewport: CSS-переменные и expand привязываем
  // в .then — сразу после запуска mount компонент ещё не смонтирован, и
  // isAvailable() у bind*/expand* вернёт false.
  if (mountMiniAppSync.isAvailable()) {
    mountMiniAppSync()
    if (bindMiniAppCssVars.isAvailable()) {
      bindMiniAppCssVars()
    }
  }
  if (mountViewport.isAvailable()) {
    void mountViewport().then(() => {
      if (bindViewportCssVars.isAvailable()) {
        bindViewportCssVars()
      }
      if (expandViewport.isAvailable()) {
        // postEvent без моста (mock-браузер, часть клиентов) бросает
        // UnknownEnvError — развернёмость экрана не должна ронять приложение.
        try {
          expandViewport()
        } catch {
          // нет моста для postEvent — остаёмся как есть
        }
      }
    })
  }
  if (mountBackButton.isAvailable()) {
    void mountBackButton()
  }

  return isTMA()
}
