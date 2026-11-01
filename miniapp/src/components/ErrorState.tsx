import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/Button"
import { apiErrorMessage } from "@/lib/errors"

/** Экранное состояние ошибки: текст по статусу + кнопка повтора. */
export function ErrorState({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  const { t } = useTranslation()

  return (
    <div role="alert" className="space-y-3 py-8 text-center">
      <p className="text-sm text-muted-foreground">{apiErrorMessage(t, error)}</p>
      <Button variant="outline" size="sm" onClick={onRetry}>
        {t("common.retry")}
      </Button>
    </div>
  )
}
