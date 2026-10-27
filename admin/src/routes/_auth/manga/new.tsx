import { createFileRoute } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

import { PlaceholderPage } from "@/components/PlaceholderPage"

export const Route = createFileRoute("/_auth/manga/new")({
  component: MangaNewPage,
})

function MangaNewPage() {
  const { t } = useTranslation()
  return <PlaceholderPage title={`${t("nav.manga")}: new`} stage="4" />
}
