import { createFileRoute, redirect, useNavigate } from "@tanstack/react-router"
import { useState } from "react"
import { useTranslation } from "react-i18next"

import { ApiError, api, unwrap } from "@/api/client"
import { queryClient } from "@/api/queryClient"
import { refreshSession, resetRefreshQueue } from "@/api/refresh"
import { Button } from "@/components/ui/Button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card"
import { Input } from "@/components/ui/Input"

export const Route = createFileRoute("/login")({
  beforeLoad: async () => {
    // Проба сессии через refresh-ручку, а не GET /sources: 401 от защищённой
    // ручки auth-middleware превращает в emitUnauthorized (тост «сессия
    // истекла» + редирект), и на /login возникает цикл: гард -> 401 -> тост ->
    // navigate -> гард. Refresh успешен только при живой refresh-cookie —
    // ровно тогда, когда логин не нужен.
    if (await refreshSession()) {
      throw redirect({ to: "/manga" })
    }
  },
  component: LoginPage,
})

function LoginPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setPending(true)
    try {
      await unwrap(await api.POST("/auth/login", { body: { username, password } }))
      // Чистый старт: остатки кеша от прошлой сессии не показываем.
      resetRefreshQueue()
      queryClient.clear()
      await navigate({ to: "/manga" })
    } catch (err) {
      setError(
        err instanceof ApiError && err.status === 401
          ? t("login.invalidCredentials")
          : t("errors.network"),
      )
    } finally {
      setPending(false)
    }
  }

  return (
    <div className="flex min-h-svh items-center justify-center p-4">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle className="text-xl">{t("login.title")}</CardTitle>
          <CardDescription>{t("app.title")}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="space-y-4">
            <div className="space-y-1.5">
              <label htmlFor="login-username" className="block text-sm font-medium">
                {t("login.username")}
              </label>
              <Input
                id="login-username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                autoComplete="username"
                required
              />
            </div>
            <div className="space-y-1.5">
              <label htmlFor="login-password" className="block text-sm font-medium">
                {t("login.password")}
              </label>
              <Input
                id="login-password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete="current-password"
                required
              />
            </div>
            {error && <p className="text-sm text-destructive">{error}</p>}
            <Button type="submit" className="w-full" disabled={pending}>
              {pending ? t("login.submitting") : t("login.submit")}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
