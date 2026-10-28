import { useTranslation } from "react-i18next"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card"

// Временная заглушка страниц этапов 3–4.
export function PlaceholderPage(props: { title: string; stage: string }) {
  const { t } = useTranslation()
  return (
    <Card>
      <CardHeader>
        <CardTitle>{props.title}</CardTitle>
      </CardHeader>
      <CardContent className="text-sm text-muted-foreground">
        {t("placeholder.inProgress", { stage: props.stage })}
      </CardContent>
    </Card>
  )
}
