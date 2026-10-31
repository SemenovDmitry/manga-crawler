import { useTranslation } from "react-i18next"

import { Card, CardContent } from "@/components/ui/Card"

/** Этап 1 — заглушка. Статичный текст справки RU/EN появится позже. */
export function Help() {
  const { t } = useTranslation()

  return (
    <section className="space-y-3">
      <h1 className="text-lg font-semibold">{t("help.title")}</h1>
      <Card>
        <CardContent className="pt-6 text-sm text-muted-foreground">
          {t("placeholder.inProgress", { stage: 3 })}
        </CardContent>
      </Card>
    </section>
  )
}
