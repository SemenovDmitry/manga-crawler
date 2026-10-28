import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { type ApiError, api, unwrap } from "@/api/client"
import { useApiToast } from "@/hooks/useApiToast"
import type { IChatUser } from "@/types/models"

const chatUsersKey = ["chat-users"] as const

export function useChatUsers() {
  return useQuery({
    queryKey: chatUsersKey,
    queryFn: async () => {
      const res = await api.GET("/chat-users")
      return unwrap(res)
    },
  })
}

// enabled=true в API соответствует !blocked в БД (ручка инвертирует).
export function useSetChatUserEnabled() {
  const qc = useQueryClient()
  const toastError = useApiToast()

  return useMutation({
    mutationFn: async ({ chatId, enabled }: { chatId: number; enabled: boolean }) => {
      const res = await api.PATCH("/chat-users/{chat_id}/enabled", {
        params: { path: { chat_id: chatId } },
        body: { enabled },
      })
      return unwrap(res)
    },
    onMutate: async ({ chatId, enabled }) => {
      await qc.cancelQueries({ queryKey: chatUsersKey })
      const prev = qc.getQueryData<{ users: IChatUser[] }>(chatUsersKey)
      if (prev) {
        qc.setQueryData<{ users: IChatUser[] }>(chatUsersKey, {
          users: prev.users.map((u) =>
            u.chat_id === chatId ? { ...u, enabled, blocked: !enabled } : u,
          ),
        })
      }
      return { prev }
    },
    onError: (err, _vars, ctx) => {
      if (ctx?.prev) {
        qc.setQueryData(chatUsersKey, ctx.prev)
      }
      toastError(err as ApiError)
    },
    onSettled: () => {
      void qc.invalidateQueries({ queryKey: chatUsersKey })
    },
  })
}
