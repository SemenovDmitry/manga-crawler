import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { api } from "@/api/client"
import type { MangaList, SubscriptionState } from "@/api/types"

/** Подписки текущего пользователя. select — сразу список манг. */
export function useSubscriptions() {
  return useQuery({
    queryKey: ["subscriptions"],
    queryFn: () => api<MangaList>("/subscriptions"),
    select: (data) => data.manga,
  })
}

/** Отписка с optimistic update: строка исчезает сразу, при ошибке —
 * откат, после — сверка с сервером (invalidate). */
export function useUnsubscribe() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (mangaId: number) =>
      api<SubscriptionState>(`/subscriptions/${mangaId}`, { method: "DELETE" }),
    onMutate: async (mangaId) => {
      await queryClient.cancelQueries({ queryKey: ["subscriptions"] })
      const previous = queryClient.getQueryData<MangaList>(["subscriptions"])
      queryClient.setQueryData<MangaList>(["subscriptions"], (old) =>
        old ? { manga: old.manga.filter((m) => m.id !== mangaId) } : old,
      )
      return { previous }
    },
    onError: (_error, _mangaId, context) => {
      if (context?.previous) {
        queryClient.setQueryData(["subscriptions"], context.previous)
      }
    },
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: ["subscriptions"] })
    },
  })
}
