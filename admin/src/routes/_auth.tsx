import { createFileRoute, Link, Outlet, redirect, useNavigate } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

import { api } from "@/api/client"
import { queryClient } from "@/api/queryClient"
import { resetRefreshQueue } from "@/api/refresh"
import { Button } from "@/components/ui/Button"
import { changeLocale, type Locale, locales } from "@/i18n"

// Аутент-гард: защищённые страницы рендерятся только после успешного
// пробного запроса. 401 -> редирект на /login (cookie ставит бэкенд).
export const Route = createFileRoute("/_auth")({
  beforeLoad: async () => {
    const { error } = await api.GET("/sources")
    if (error) {
      throw redirect({ to: "/login" })
    }
  },
  component: AuthLayout,
})

function AuthLayout() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  async function onLogout() {
    // Сначала сбрасываем состояние сессии на клиенте: очередь refresh и кеш
    // данных (после перелогина всё должно загрузиться заново). Сам logout
    // идёт с действующей access-cookie; при её истечении получим 401 —
    // refresh уже погашен (refresh-cookie отозвана бэкендом), редирект на
    // логин произойдёт и так.
    resetRefreshQueue()
    queryClient.clear()
    await api.POST("/auth/logout")
    await navigate({ to: "/login" })
  }

  return (
    <div className="min-h-svh">
      <header className="border-b bg-sidebar">
        <div className="mx-auto flex h-14 max-w-6xl items-center gap-6 px-4">
          <nav className="flex items-center gap-4 text-sm">
            <AppLink to="/manga">{t("nav.manga")}</AppLink>
            <AppLink to="/sources">{t("nav.sources")}</AppLink>
            <AppLink to="/chat-users">{t("nav.chatUsers")}</AppLink>
          </nav>
          <div className="ml-auto flex items-center gap-2">
            <LocaleSwitch />
            <Button variant="ghost" size="sm" onClick={onLogout}>
              {t("nav.logout")}
            </Button>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-6xl p-4">
        <Outlet />
      </main>
    </div>
  )
}

function AppLink(props: { to: "/manga" | "/sources" | "/chat-users"; children: React.ReactNode }) {
  // TanStack Link: типизированный to из routeTree, полноценная SPA-навигация,
  // активный маршрут — подсветка через activeProps.
  return (
    <Link
      to={props.to}
      className="text-sidebar-foreground/80 hover:text-sidebar-foreground"
      activeProps={{ className: "text-sidebar-foreground font-medium" }}
    >
      {props.children}
    </Link>
  )
}

function LocaleSwitch() {
  const { t } = useTranslation()
  return (
    <label className="flex items-center gap-1 text-xs text-muted-foreground">
      <span>{t("common.language")}</span>
      <select
        className="rounded-md border border-input bg-transparent px-1.5 py-1 text-xs"
        defaultValue={localStorage.getItem("admin-locale") ?? "ru"}
        onChange={(e) => changeLocale(e.target.value as Locale)}
      >
        {locales.map((l) => (
          <option key={l} value={l}>
            {l.toUpperCase()}
          </option>
        ))}
      </select>
    </label>
  )
}
