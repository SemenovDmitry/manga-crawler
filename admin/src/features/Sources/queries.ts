import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { type ApiError, api, unwrap } from "@/api/client"
import { useApiToast } from "@/hooks/useApiToast"
import type { ISource  } from "@/types/models"

const sourcesKey = ["sources"] as const

export function useSources() {
  return useQuery({
    queryKey: sourcesKey,
    queryFn: async () => {
      const res = await api.GET("/sources")
      return unwrap(res)
    },
  })
}

function useSourceErrorToast() {
  const toast = useApiToast()
  return (err: unknown) => {
    // 404 источника после параллельного удаления — частный случай; для
    // остальных текст даст errorMessages (409/5xx → i18n, остальное — бэкенд).
    toast(err)
  }
}

// Включение/выключение источника — optimistic: свитч реагирует мгновенно,
// при ошибке значение откатывается и показывается тост.
export function useSetSourceEnabled() {
  const qc = useQueryClient()
  const toastError = useSourceErrorToast()

  return useMutation({
    mutationFn: async ({ id, enabled }: { id: number; enabled: boolean }) => {
      const res = await api.PATCH("/sources/{id}/enabled", {
        params: { path: { id } },
        body: { enabled },
      })
      return unwrap(res)
    },
    onMutate: async ({ id, enabled }) => {
      await qc.cancelQueries({ queryKey: sourcesKey })
      const prev = qc.getQueryData<{ sources: ISource[] }>(sourcesKey)
      if (prev) {
        qc.setQueryData<{ sources: ISource[] }>(sourcesKey, {
          sources: prev.sources.map((s) => (s.id === id ? { ...s, enabled } : s)),
        })
      }
      return { prev }
    },
    onError: (err, _vars, ctx) => {
      if (ctx?.prev) {
        qc.setQueryData(sourcesKey, ctx.prev)
      }
      toastError(err as ApiError)
    },
    onSettled: () => {
      void qc.invalidateQueries({ queryKey: sourcesKey })
    },
  })
}

// Сброс полного обхода — по кнопке с подтверждением; optimistic не нужен.
export function useResetSourceFullCrawl() {
  const qc = useQueryClient()
  const toastError = useSourceErrorToast()

  return useMutation({
    mutationFn: async (id: number) => {
      const res = await api.POST("/sources/{id}/recrawl", { params: { path: { id } } })
      return unwrap(res)
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: sourcesKey })
    },
    onError: (err) => toastError(err as ApiError),
  })
}
