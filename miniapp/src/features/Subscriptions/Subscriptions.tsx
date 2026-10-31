import { useTranslation } from "react-i18next"

import { Card, CardContent } from "@/components/ui/Card"

/** Этап 1 — заглушка. Список подписок с optimistic update появится на этапе 3. */
export function Subscriptions() {
  const { t } = useTranslation()

  return (
    <section className="space-y-3">
      <h1 className="text-lg font-semibold">{t("subscriptions.title")}</h1>
      <Card>
        <CardContent className="pt-6 text-sm text-muted-foreground">
          {t("subscriptions.empty")}
        </CardContent>
      </Card>
    </section>
  )
}
