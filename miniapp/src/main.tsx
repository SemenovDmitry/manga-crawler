import { QueryClientProvider } from "@tanstack/react-query"
import { StrictMode } from "react"
import { createRoot } from "react-dom/client"
import { App } from "@/App"
import { queryClient } from "@/api/queryClient"
import { bootstrapTelegram } from "@/lib/telegram"
import "@/i18n"
import "@/styles/globals.css"

const root = document.getElementById("root")
if (!root) {
  throw new Error("#root не найден")
}

// StrictMode вызывает эффекты дважды — bootstrap идемпотентен (см. telegram.ts).
bootstrapTelegram()

createRoot(root).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </StrictMode>,
)
