import { createRouter } from "@tanstack/react-router"

import { queryClient } from "@/api/queryClient"
import { routeTree } from "./routeTree.gen"

export const router = createRouter({
  routeTree,
  context: { queryClient },
  defaultPreload: "intent",
  // 401 из API трактуем глобально: не кешируем, редирект на /login делает guard.
  defaultPreloadStaleTime: 0,
  scrollRestoration: true,
})

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router
  }
}
