import type { QueryClient } from "@tanstack/react-query"
import { createRootRouteWithContext, Link, Outlet, useNavigate } from "@tanstack/react-router"
import { useEffect } from "react"
import { useTranslation } from "react-i18next"

import { onUnauthorized } from "@/api/authEvents"
import { Toaster } from "@/components/Toaster"
import { showToast } from "@/lib/toastStore"
export interface RouterContext {
  queryClient: QueryClient
}

export const Route = createRootRouteWithContext<RouterContext>()({
  component: RootLayout,
  notFoundComponent: NotFound,
})

function RootLayout() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  // Refresh не удался -> тост и редирект на логин. Подписка один раз на всё
  // приложение; чистка кеша гарантирует, что после перелогина данные
  // перезагрузятся, а не возьмутся из стейла.
  useEffect(
    () =>
      onUnauthorized(() => {
        showToast("error", t("errors.unauthorized"))
        void navigate({ to: "/login" })
      }),
    [navigate, t],
  )

  return (
    <>
      <Outlet />
      <Toaster />
    </>
  )
}

function NotFound() {
  const { t } = useTranslation()
  return (
    <div className="flex min-h-svh flex-col items-center justify-center gap-4">
      <p className="text-4xl font-semibold">404</p>
      <Link to="/" className="text-primary underline-offset-4 hover:underline">
        {t("common.back")}
      </Link>
    </div>
  )
}
