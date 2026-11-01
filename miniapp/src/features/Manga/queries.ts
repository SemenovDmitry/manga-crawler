import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { api } from "@/api/client"
import type { MangaDetail, SubscriptionState } from "@/api/types"

const detailKey = (mangaId: string) => ["manga", "detail", mangaId] as const

/** Карточка манги + признак подписки текущего пользователя. */
export function useManga(mangaId: string) {
  return useQuery({
    queryKey: detailKey(mangaId),
    queryFn: () => api<MangaDetail>(`/manga/${mangaId}`),
  })
}

/** Подписка/отписка из карточки: флаг subscribed меняется оптимистично,
 * список подписок после — инвалидируется. */
function useSetSubscribed(mangaId: string, subscribe: boolean) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () =>
      api<SubscriptionState>(`/subscriptions/${mangaId}`, {
        method: subscribe ? "PUT" : "DELETE",
      }),
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: detailKey(mangaId) })
      const previous = queryClient.getQueryData<MangaDetail>(detailKey(mangaId))
      queryClient.setQueryData<MangaDetail>(detailKey(mangaId), (old) =>
        old ? { ...old, subscribed: subscribe } : old,
      )
      return { previous }
    },
    onError: (_error, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(detailKey(mangaId), context.previous)
      }
    },
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: detailKey(mangaId) })
      void queryClient.invalidateQueries({ queryKey: ["subscriptions"] })
    },
  })
}

export function useSubscribe(mangaId: string) {
  return useSetSubscribed(mangaId, true)
}

export function useUnsubscribe(mangaId: string) {
  return useSetSubscribed(mangaId, false)
}
