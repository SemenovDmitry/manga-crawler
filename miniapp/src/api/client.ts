/**
 * HTTP-клиент miniapi. Аутентификация — Telegram initData в заголовке
 * каждого запроса (stateless, без cookie и сессий); проверку подписи делает
 * miniapi (этап 2).
 *
 * База: VITE_API_BASE_URL (прод — полный URL miniapi); по умолчанию пусто —
 * запросы идут на относительный /mini-api/... через Vite-прокси в dev.
 */
import { initData } from "@telegram-apps/sdk-react"

export class ApiError extends Error {
  readonly status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = "ApiError"
    this.status = status
  }
}

const baseUrl = (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? ""

/** Сырой initData-квиток; в dev-мок-режиме подставляется mock-значение. */
function rawInitData(): string {
  try {
    return initData.raw() ?? ""
  } catch {
    return ""
  }
}

export async function api<T>(
  path: string,
  options: { method?: string; body?: unknown } = {},
): Promise<T> {
  const raw = rawInitData()
  if (!raw) {
    // Вне Telegram и без мока данных нет — запрос бессмыслен.
    throw new ApiError(401, "no initData")
  }

  const headers: Record<string, string> = {
    "X-Telegram-Init-Data": raw,
  }
  if (options.body !== undefined) {
    headers["Content-Type"] = "application/json"
  }

  const response = await fetch(`${baseUrl}/mini-api${path}`, {
    method: options.method ?? "GET",
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
  })

  if (!response.ok) {
    let message = `HTTP ${response.status}`
    try {
      const data = (await response.json()) as { message?: string; error?: string }
      message = data.message ?? data.error ?? message
    } catch {
      // тело не JSON — оставляем статусное сообщение
    }
    throw new ApiError(response.status, message)
  }

  return (await response.json()) as T
}
