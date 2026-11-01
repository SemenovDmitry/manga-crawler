import { useCallback } from "react"
import { useTranslation } from "react-i18next"

import { BottomNav } from "@/components/BottomNav"
import Help from "@/features/Help"
import MangaDetail from "@/features/Manga"
import Search from "@/features/Search"
import Subscriptions from "@/features/Subscriptions"
import { useBackButton } from "@/hooks/useBackButton"
import { useRoute } from "@/lib/router"

export function App() {
  const { t } = useTranslation()
  const route = useRoute()

  // Системная «Назад» Telegram активна только на вложенной странице — карточке.
  const goBack = useCallback(() => {
    window.history.back()
  }, [])

  useBackButton(route.name === "manga", goBack)

  return (
    <div className="mx-auto flex min-h-svh w-full max-w-lg flex-col">
      <main className="flex-1 px-4 pt-4 pb-6">
        {route.name === "search" && <Search />}
        {route.name === "subscriptions" && <Subscriptions />}
        {route.name === "help" && <Help />}
        {route.name === "manga" && <MangaDetail mangaId={route.mangaId} />}
      </main>
      {/* Карточка манги — вложенный экран, нижней навигации не имеет */}
      {route.name !== "manga" && <BottomNav active={route.name} />}
      <span className="sr-only">{t("app.title")}</span>
    </div>
  )
}
