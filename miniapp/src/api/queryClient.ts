import { QueryClient } from "@tanstack/react-query"

/**
 * Общий query-клиент miniapp. 401 (истёк initData) не ретраится — все
 * экраны уходят в ошибку; переоткрытие мини-аппа даёт свежий initData.
 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: (failureCount, error) => {
        // Клиентские 4xx (401 и прочие) не ретраим.
        const status = (error as { status?: number }).status
        if (typeof status === "number" && status >= 400 && status < 500) {
          return false
        }
        return failureCount < 2
      },
    },
  },
})
