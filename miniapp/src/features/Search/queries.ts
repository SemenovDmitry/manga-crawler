import { keepPreviousData, useQuery } from "@tanstack/react-query"

import { api } from "@/api/client"
import type { MangaList } from "@/api/types"

const SEARCH_LIMIT = 20

/** Поиск по названию; при пустом запросе не выполняется. keepPreviousData —
 * во время набора текста показываются результаты прошлого запроса. */
export function useMangaSearch(query: string) {
  return useQuery({
    queryKey: ["manga", "search", query],
    queryFn: () =>
      api<MangaList>(`/manga/search?q=${encodeURIComponent(query)}&limit=${SEARCH_LIMIT}`),
    enabled: query.trim() !== "",
    placeholderData: keepPreviousData,
  })
}
