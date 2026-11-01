import type { TFunction } from "i18next"

import { ApiError } from "@/api/client"

/** Человекочитаемый текст ошибки API по её статусу; прочие ошибки (упавший
 * fetch — это сеть) считаем проблемой связи. */
export function apiErrorMessage(t: TFunction<"translation", undefined>, error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 401) return t("errors.unauthorized")
    if (error.status === 404) return t("errors.notFound")
    if (error.status >= 500) return t("errors.server")
    return t("errors.unknown")
  }
  return t("errors.network")
}
