import { ApiError } from "@/api/client"

// Ошибки бэкенда — русские строки {error}; интерфейс мультиязычный, поэтому
// известные коды статусов маппим на локализованные сообщения, а неизвестные
// показываем как есть (текст от бэкенда).

const statusMessages = {
  unauthorized: "errors.unauthorized",
  forbidden: "errors.forbidden",
  notFound: "errors.notFound",
  conflict: "errors.conflict",
  server: "errors.server",
} as const

const statusKeys: Record<number, keyof typeof statusMessages> = {
  401: "unauthorized",
  403: "forbidden",
  404: "notFound",
  409: "conflict",
  500: "server",
  502: "server",
  503: "server",
}

// i18n-ключ или прямой текст ошибки.
export function errorToMessage(err: unknown): { key?: string; text?: string } {
  if (err instanceof ApiError) {
    const mapped = statusKeys[err.status]
    if (mapped) {
      return { key: statusMessages[mapped] }
    }
    if (err.message) {
      return { text: err.message }
    }
  }
  if (err instanceof TypeError) {
    // fetch не дошёл до сервера (нет сети / сервер выключен).
    return { key: "errors.network" }
  }
  return { key: "errors.unknown" }
}
