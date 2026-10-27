import { QueryClient } from "@tanstack/react-query"

// 401 -> глобальный обработчик (root route) запускает refresh и повтор;
// здесь только дефолты кеширования.
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: (failureCount, error) => {
        // Ошибки авторизации и клиентские 4xx не ретраим.
        const status = (error as { status?: number }).status
        if (typeof status === "number" && status >= 400 && status < 500) {
          return false
        }
        return failureCount < 2
      },
    },
  },
})
