import { createFileRoute, Link, Outlet, redirect, useNavigate } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

import { api } from "@/api/client"
import { queryClient } from "@/api/queryClient"
import { resetRefreshQueue } from "@/api/refresh"
import { LocaleSwitch } from "@/components/LocaleSwitch"
import { Button } from "@/components/ui/Button"

export const Route = createFileRoute("/_auth")({
  beforeLoad: async () => {
    // Проверка сессии при входе на защищённые страницы; её 401 при мёртвой
    // сессии middleware также превращает в тост + редирект — здесь редирект
    // просто дублирует его на случай прямого захода по URL.
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
    // Сбрасываем очередь refresh, затем гасим сессию на бэкенде и только
    // после перехода чистим кеш запросов: clear() при смонтированных
    // обсерверах рождает волну рефетчей, которые после logout дают 401 и
    // шторм тостов «сессия истекла». На /login наблюдателей нет — очистка
    // тихая. Сбой logout не мешаем показывать: без cookie защищённые
    // страницы всё равно не открыть.
    resetRefreshQueue()
    await api.POST("/auth/logout").catch(() => undefined)
    await navigate({ to: "/login", replace: true })
    queryClient.clear()
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
