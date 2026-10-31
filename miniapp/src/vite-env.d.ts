/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Полный URL miniapi на проде; в dev не задан — работает Vite-прокси. */
  readonly VITE_API_BASE_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
