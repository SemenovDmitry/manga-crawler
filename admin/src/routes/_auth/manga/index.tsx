import { createFileRoute } from "@tanstack/react-router"

import Manga from "@/features/Manga"
import { mangaSearchSchema } from "@/features/Manga/search"

// Фильтры каталога живут в URL: validateSearch валидирует и типизирует их.
export const Route = createFileRoute("/_auth/manga/")({
  validateSearch: mangaSearchSchema,
  component: MangaListPage,
})

function MangaListPage() {
  const search = Route.useSearch()
  const navigate = Route.useNavigate()

  return (
    <Manga
      search={search}
      onSearchChange={(patch) => {
        void navigate({ search: (prev) => ({ ...prev, ...patch }) })
      }}
    />
  )
}
