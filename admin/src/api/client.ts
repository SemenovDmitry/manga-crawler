import createClient, { type Middleware } from "openapi-fetch"

import { emitUnauthorized } from "./authEvents"
import { refreshSession } from "./refresh"
import type { paths } from "./schema.gen"

// Клиент openapi-fetch: типы запросов/ответов выводятся из docs/openapi.yaml
// (файл генерируется `npm run gen:api`). Cookie браузер добавляет сам —
// токены в JS не живут. same-origin: в dev это прокси Vite.
//
// Мидлварь ниже перехватывает 401 от защищённых ручек: один общий refresh
// (см. refresh.ts — очередь дедуплицирует параллельные 401) и ровно один
// повтор исходного запроса. Повторный 401 после успешного refresh — ошибка
// авторизации, наружу через emitUnauthorized.

const authMiddleware: Middleware = {
  async onResponse({ request, response }) {
    // 401 login-ручки (неверный пароль) — обычный ответ, не сигнал refresh.
    const isAuthPath = request.url.includes("/auth/login")
    if (response.status !== 401 || isAuthPath) {
      return response
    }

    const refreshed = await refreshSession()
    if (!refreshed) {
      emitUnauthorized()
      return response
    }

    return fetch(request)
  },
}

export const api = createClient<paths>({
  baseUrl: "/admin-api",
  // openapi-fetch передаёт готовый Request; no-store — админ-API отдаёт
  // живые данные, браузерный кеш здесь только мешает.
  fetch: (input) => fetch(input, { cache: "no-store" }),
})

api.use(authMiddleware)

// Ошибка бэкенда: {error: "человеческий текст"}.
export class ApiError extends Error {
  readonly status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = "ApiError"
    this.status = status
  }
}

// unwrap — общий разбор ответа openapi-fetch: ошибка бэкенда -> ApiError,
// 204 -> undefined.
export async function unwrap<T>(response: {
  data?: T
  error?: unknown
  response: Response
}): Promise<T> {
  if (response.error !== undefined && response.error !== null) {
    const message =
      typeof response.error === "object" &&
      response.error !== null &&
      "error" in response.error &&
      typeof (response.error as { error: unknown }).error === "string"
        ? (response.error as { error: string }).error
        : `HTTP ${response.response.status}`
    throw new ApiError(response.response.status, message)
  }
  if (response.response.status === 204) {
    return undefined as T
  }
  if (response.data === undefined) {
    throw new ApiError(response.response.status, `HTTP ${response.response.status}`)
  }
  return response.data
}
