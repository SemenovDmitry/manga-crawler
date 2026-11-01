import path from "node:path"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

// base — деплой в подпуть GitHub Pages этого репозитория
// (https://SemenovDmitry.github.io/manga-crawler/).
// Прокси /mini-api → miniapi (:8082) — как в admin, CORS в dev не нужен.
export default defineConfig({
  base: "/manga-crawler/",
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    port: 5174,
    proxy: {
      "/mini-api": {
        target: "http://localhost:8082",
        changeOrigin: true,
      },
    },
  },
})
