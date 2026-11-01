import { useTranslation } from "react-i18next"

import type { PublicationStatus } from "@/api/types"
import { Badge } from "@/components/ui/Badge"

const STATUS_VARIANTS: Record<PublicationStatus, "default" | "secondary" | "outline"> = {
  ongoing: "default",
  completed: "secondary",
  hiatus: "outline",
  cancelled: "outline",
  discontinued: "outline",
}

/** Статус публикации манги — бейдж с локализованной подписью. */
export function StatusBadge({ status }: { status: PublicationStatus }) {
  const { t } = useTranslation()
  return <Badge variant={STATUS_VARIANTS[status]}>{t(`manga.status.${status}`)}</Badge>
}
