import { type ReactNode, useState } from "react"
import { useTranslation } from "react-i18next"

import type { Manga } from "@/api/types"
import { ErrorState } from "@/components/ErrorState"
import { MangaRow } from "@/components/MangaRow"
import { Input } from "@/components/ui/Input"
import { Skeleton } from "@/components/ui/Skeleton"
import { useDebouncedValue } from "@/hooks/useDebouncedValue"
import { navigate } from "@/lib/router"
import { useMangaSearch } from "./queries"

const DEBOUNCE_MS = 400

const SKELETON_IDS = ["s1", "s2", "s3", "s4", "s5"]

function SearchSkeleton() {
  return (
    <div className="space-y-2" aria-hidden>
      {SKELETON_IDS.map((id) => (
        <div key={id} className="flex items-center gap-3 p-2">
          <Skeleton className="size-10 rounded-md" />
          <div className="flex-1 space-y-2">
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-3 w-1/2" />
          </div>
        </div>
      ))}
    </div>
  )
}

function SearchResults({ manga }: { manga: Manga[] }) {
  return (
    <div className="divide-y">
      {manga.map((m) => (
        <MangaRow
          key={m.id}
          manga={m}
          onClick={() => navigate({ name: "manga", mangaId: String(m.id) })}
        />
      ))}
    </div>
  )
}

/** Поиск манги: поле с debounce-запросом и список строк-результатов. */
export function Search() {
  const { t } = useTranslation()
  const [query, setQuery] = useState("")
  const debouncedQuery = useDebouncedValue(query.trim(), DEBOUNCE_MS)
  const search = useMangaSearch(debouncedQuery)

  let body: ReactNode
  if (debouncedQuery === "") {
    body = <p className="text-sm text-muted-foreground">{t("search.emptyQuery")}</p>
  } else if (search.isPending) {
    body = <SearchSkeleton />
  } else if (search.isError) {
    body = <ErrorState error={search.error} onRetry={() => void search.refetch()} />
  } else if (search.data.manga.length === 0) {
    body = <p className="text-sm text-muted-foreground">{t("search.empty")}</p>
  } else {
    body = <SearchResults manga={search.data.manga} />
  }

  return (
    <section className="space-y-3">
      <h1 className="text-lg font-semibold">{t("search.title")}</h1>
      <Input
        type="search"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder={t("search.placeholder")}
      />
      {body}
    </section>
  )
}
