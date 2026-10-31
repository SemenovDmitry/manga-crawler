import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/Badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card"

/** Этап 1 — заглушка карточки. Данные манги и кнопка подписки — этап 3. */
export function MangaDetail({ mangaId }: { mangaId: string }) {
  const { t } = useTranslation()

  return (
    <section className="space-y-3">
      <Card>
        <CardHeader>
          <CardTitle>{t("manga.notFound")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          <Badge variant="outline">{mangaId}</Badge>
          <p className="text-sm text-muted-foreground">
            {t("placeholder.inProgress", { stage: 3 })}
          </p>
        </CardContent>
      </Card>
    </section>
  )
}
