import { useTranslation } from "react-i18next"

import { Input } from "@/components/ui/Input"

/** Этап 1 — заглушка. Живой поиск с debounce появится на этапе 3. */
export function Search() {
  const { t } = useTranslation()

  return (
    <section className="space-y-3">
      <h1 className="text-lg font-semibold">{t("search.title")}</h1>
      <Input placeholder={t("search.placeholder")} />
      <p className="text-sm text-muted-foreground">{t("search.emptyQuery")}</p>
    </section>
  )
}
