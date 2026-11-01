import { useSyncExternalStore } from "react"

/**
 * Минимальный hash-роутер (GitHub Pages не умеет rewrite на index.html).
 * Маршруты MVP: поиск, подписки, справка, карточка манги.
 */

export type Route =
  | { name: "search" }
  | { name: "subscriptions" }
  | { name: "help" }
  | { name: "manga"; mangaId: string }

export const HOME_ROUTE: Route = { name: "subscriptions" }

export function parseHash(rawHash: string): Route {
  const segments = rawHash.replace(/^#/, "").split("/").filter(Boolean)
  switch (segments[0]) {
    case "search":
      return { name: "search" }
    case "subscriptions":
      return { name: "subscriptions" }
    case "help":
      return { name: "help" }
    case "manga": {
      const mangaId = segments[1]
      return mangaId ? { name: "manga", mangaId: decodeURIComponent(mangaId) } : HOME_ROUTE
    }
    default:
      return HOME_ROUTE
  }
}

export function routeToHash(route: Route): string {
  switch (route.name) {
    case "search":
      return "#/search"
    case "subscriptions":
      return "#/subscriptions"
    case "help":
      return "#/help"
    case "manga":
      return `#/manga/${encodeURIComponent(route.mangaId)}`
  }
}

/** Программная навигация: меняет location.hash, дальше сработает hashchange. */
export function navigate(route: Route) {
  window.location.hash = routeToHash(route)
}

/**
 * Deep link на мини-апп `t.me/<bot>?startapp=<manga_id>`: Telegram передаёт
 * параметр в launch-параметре `tgWebAppStartParam` (оказывается в query
 * загруженной страницы). `startapp` — то же самое, удобно в dev-браузере.
 * Вызывается до первого рендера, чтобы приложение открылось сразу
 * на карточке; нечисловой параметр игнорируется — останется стартовый экран.
 */
export function applyStartDeepLink(): void {
  const params = new URLSearchParams(window.location.search)
  const startParam = params.get("tgWebAppStartParam") ?? params.get("startapp")
  if (!startParam || !/^\d+$/.test(startParam)) {
    return
  }
  navigate({ name: "manga", mangaId: startParam })
}

function subscribe(onChange: () => void) {
  window.addEventListener("hashchange", onChange)
  return () => window.removeEventListener("hashchange", onChange)
}

function getSnapshot() {
  return window.location.hash
}

export function useRoute(): Route {
  const hash = useSyncExternalStore(subscribe, getSnapshot)
  return parseHash(hash)
}
