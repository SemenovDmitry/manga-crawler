import { createFileRoute, redirect } from "@tanstack/react-router"

import { refreshSession } from "@/api/refresh"
import Login from "@/features/Login"

export const Route = createFileRoute("/login")({
  beforeLoad: async () => {
    // Обратный гард страницы логина (защищённые страницы охраняет _auth.tsx).
    // Проба сессии через refresh-ручку, а не GET /sources: 401 от защищённой
    // ручки auth-middleware превращает в emitUnauthorized (тост «сессия
    // истекла» + редирект), и на /login возникает цикл: гард -> 401 -> тост ->
    // navigate -> гард. Refresh успешен только при живой refresh-cookie —
    // ровно тогда, когда логин не нужен.
    if (await refreshSession()) {
      throw redirect({ to: "/manga" })
    }
  },
  component: Login,
})
