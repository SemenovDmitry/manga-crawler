import path from "node:path"
import tailwindcss from "@tailwindcss/vite"
import { tanstackRouter } from "@tanstack/router-plugin/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

// base './' — статика после сборки работает из любого места (подготовка
// к будущему go:embed). Прокси /admin-api → adminapi (:8081): cookie
// SameSite=Strict ходят в одном домене, CORS на бэкенде не нужен.
export default defineConfig({
  base: "./",
  plugins: [
    // router-плагин до react — генерирует routeTree.gen.ts из src/routes.
    // tanstackRouter = generator + codeSplitter (TanStackRouterVite deprecated).
    tanstackRouter({ target: "react", autoCodeSplitting: true }),
    react(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/admin-api": {
        target: "http://localhost:8081",
        changeOrigin: true,
      },
    },
  },
})
