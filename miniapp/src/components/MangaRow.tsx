import { BookMarked } from "lucide-react"
import type { ReactNode } from "react"
import { useTranslation } from "react-i18next"

import type { Manga } from "@/api/types"
import { StatusBadge } from "@/components/StatusBadge"

/** Строка манги в списках (поиск, подписки). Обложек в каталоге нет —
 * плейсхолдер-плитка с иконкой; action — дополнительная кнопка справа. */
export function MangaRow({
  manga,
  onClick,
  action,
}: {
  manga: Manga
  onClick: () => void
  action?: ReactNode
}) {
  const { t } = useTranslation()

  return (
    <div className="flex items-center gap-1">
      <button
        type="button"
        onClick={onClick}
        className="flex min-w-0 flex-1 cursor-pointer items-center gap-3 rounded-lg p-2 text-left transition-colors hover:bg-muted/60"
      >
        <span
          aria-hidden
          className="flex size-10 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground"
        >
          <BookMarked className="size-5" />
        </span>
        <span className="min-w-0 flex-1">
          <span className="block truncate text-sm font-medium">{manga.title}</span>
          <span className="mt-1 flex items-center gap-2 text-xs text-muted-foreground">
            <StatusBadge status={manga.publication_status} />
            <span className="truncate">{t("manga.chapters", { count: manga.chapters_count })}</span>
          </span>
        </span>
      </button>
      {action}
    </div>
  )
}
