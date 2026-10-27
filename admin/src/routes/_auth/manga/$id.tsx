import { createFileRoute } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

import { PlaceholderPage } from "@/components/PlaceholderPage"

export const Route = createFileRoute("/_auth/manga/$id")({
  component: MangaDetailPage,
})

function MangaDetailPage() {
  const { id } = Route.useParams()
  const { t } = useTranslation()
  return <PlaceholderPage title={`${t("nav.manga")} #${id}`} stage="4" />
}
