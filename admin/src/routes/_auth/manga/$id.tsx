import { createFileRoute } from "@tanstack/react-router"

import { MangaDetail } from "@/features/Manga"

export const Route = createFileRoute("/_auth/manga/$id")({
  component: MangaDetailPage,
})

function MangaDetailPage() {
  const { id } = Route.useParams()
  return <MangaDetail id={Number(id)} />
}
