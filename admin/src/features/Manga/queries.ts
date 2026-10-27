import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { type ApiError, api, unwrap } from "@/api/client"
import { useApiToast } from "@/hooks/useApiToast"
import type { IMangaCreate, IMangaUpdate } from "@/types/models"
import type { MangaSearch } from "./search"

function mangaListKey(params: MangaSearch) {
  return ["manga", params] as const
}

// Список манги: фильтры -> query-параметры listManga (limit/offset из page).
export function useMangaList(params: MangaSearch) {
  return useQuery({
    queryKey: mangaListKey(params),
    queryFn: async () => {
      const res = await api.GET("/manga", {
        params: {
          query: {
            query: params.query,
            source_id: params.sourceId,
            status: params.status,
            limit: params.limit,
            offset: (params.page - 1) * params.limit,
          },
        },
      })
      return unwrap(res)
    },
  })
}

export function useManga(id: number) {
  return useQuery({
    queryKey: ["manga", "detail", id] as const,
    queryFn: async () => {
      const res = await api.GET("/manga/{id}", { params: { path: { id } } })
      return unwrap(res)
    },
    enabled: Number.isFinite(id),
  })
}

// Инвалидация по префиксу ["manga"]: и список (все фильтры), и карточки.
function useMangaInvalidation() {
  const qc = useQueryClient()
  return () => {
    void qc.invalidateQueries({ queryKey: ["manga"] })
  }
}

export function useCreateManga() {
  const invalidate = useMangaInvalidation()
  const toastError = useApiToast()

  return useMutation({
    mutationFn: async (body: IMangaCreate) => {
      const res = await api.POST("/manga", { body })
      return unwrap(res)
    },
    onSuccess: invalidate,
    onError: (err) => toastError(err as ApiError),
  })
}

export function useUpdateManga() {
  const invalidate = useMangaInvalidation()
  const toastError = useApiToast()

  return useMutation({
    mutationFn: async ({ id, patch }: { id: number; patch: IMangaUpdate }) => {
      const res = await api.PATCH("/manga/{id}", { params: { path: { id } }, body: patch })
      return unwrap(res)
    },
    onSuccess: invalidate,
    onError: (err) => toastError(err as ApiError),
  })
}

export function useDeleteManga() {
  const invalidate = useMangaInvalidation()
  const toastError = useApiToast()

  return useMutation({
    mutationFn: async (id: number) => {
      const res = await api.DELETE("/manga/{id}", { params: { path: { id } } })
      return unwrap(res)
    },
    onSuccess: invalidate,
    onError: (err) => toastError(err as ApiError),
  })
}
