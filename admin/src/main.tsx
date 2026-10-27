import { StrictMode } from "react"
import { createRoot } from "react-dom/client"

import "@/i18n"
import { QueryClientProvider } from "@tanstack/react-query"
import { RouterProvider } from "@tanstack/react-router"
import { queryClient } from "@/api/queryClient"
import { router } from "@/router"

import "@/styles/globals.css"

const root = document.getElementById("root")
if (!root) {
  throw new Error("#root не найден")
}

createRoot(root).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  </StrictMode>,
)
