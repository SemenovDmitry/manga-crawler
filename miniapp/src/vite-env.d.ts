/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Полный URL miniapi на проде; в dev не задан — работает Vite-прокси. */
  readonly VITE_API_BASE_URL?: string
  /** Валидный dev-квиток initData (scripts/dev-initdata.mjs → .env.local). */
  readonly VITE_DEV_INIT_DATA?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
