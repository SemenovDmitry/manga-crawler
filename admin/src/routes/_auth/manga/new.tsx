import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"
import { MangaForm } from "@/features/Manga"
import { useCreateManga } from "@/features/Manga/queries"
import { useSources } from "@/features/Sources"
import { showToast } from "@/lib/toastStore"

export const Route = createFileRoute("/_auth/manga/new")({
  component: MangaNewPage,
})

function MangaNewPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const createManga = useCreateManga()
  const { data: sourcesData } = useSources()

  return (
    <MangaForm
      mode="create"
      sources={sourcesData?.sources ?? []}
      pending={createManga.isPending}
      onSubmit={async (data) => {
        const created = await createManga.mutateAsync(data)
        showToast("success", t("manga.created"))
        await navigate({ to: "/manga/$id", params: { id: String(created.id) } })
      }}
      onCancel={() => navigate({ to: "/manga" })}
    />
  )
}
