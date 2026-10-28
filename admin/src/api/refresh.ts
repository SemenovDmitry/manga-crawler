// Очередь обновления access-cookie при 401.
//
// Access-токен живёт 15 минут; когда он истекает, несколько параллельных
// запросов получают 401 одновременно. Refresh-ручка бэкенда одноразовая
// (ротация: использованный refresh гасится), поэтому запускаем ровно один
// refresh, остальные 401 ждут его и после успеха повторяются.

let refreshPromise: Promise<boolean> | null = null

// POST /auth/refresh: бэкенд читает refresh-cookie и ротирует пару.
// true — обновилось, можно повторить исходный запрос; false — refresh не
// удался (нет/просрочена cookie), нужен переход на логин.
export function refreshSession(): Promise<boolean> {
  refreshPromise ??= doRefresh()
  return refreshPromise
}

async function doRefresh(): Promise<boolean> {
  try {
    const res = await fetch("/admin-api/auth/refresh", { method: "POST" })
    return res.ok
  } catch {
    return false
  } finally {
    // Очередь разряжается: следующий 401 после завершения этого refresh
    // запустит новую попытку.
    refreshPromise = null
  }
}

// Тесты и принудительный сброс состояния (после logout).
export function resetRefreshQueue() {
  refreshPromise = null
}
