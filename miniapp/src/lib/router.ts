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
