import { BookmarkX } from "lucide-react"
import type { ReactNode } from "react"
import { useTranslation } from "react-i18next"

import { ErrorState } from "@/components/ErrorState"
import { MangaRow } from "@/components/MangaRow"
import { Button } from "@/components/ui/Button"
import { Skeleton } from "@/components/ui/Skeleton"
import { navigate } from "@/lib/router"
import { useSubscriptions, useUnsubscribe } from "./queries"

const SKELETON_IDS = ["s1", "s2", "s3", "s4"]

function SubscriptionsSkeleton() {
  return (
    <div className="space-y-2" aria-hidden>
      {SKELETON_IDS.map((id) => (
        <div key={id} className="flex items-center gap-3 p-2">
          <Skeleton className="size-10 rounded-md" />
          <div className="flex-1 space-y-2">
            <Skeleton className="h-4 w-2/3" />
            <Skeleton className="h-3 w-1/3" />
          </div>
        </div>
      ))}
    </div>
  )
}

/** Мои подписки: список с кнопкой отписки (optimistic update). */
export function Subscriptions() {
  const { t } = useTranslation()
  const subscriptions = useSubscriptions()
  const unsubscribe = useUnsubscribe()

  let body: ReactNode
  if (subscriptions.isPending) {
    body = <SubscriptionsSkeleton />
  } else if (subscriptions.isError) {
    body = <ErrorState error={subscriptions.error} onRetry={() => void subscriptions.refetch()} />
  } else if (subscriptions.data.length === 0) {
    body = (
      <div className="space-y-1 py-4 text-center">
        <p className="text-sm text-muted-foreground">{t("subscriptions.empty")}</p>
        <p className="text-sm text-muted-foreground">{t("subscriptions.emptyHint")}</p>
      </div>
    )
  } else {
    body = (
      <div className="divide-y">
        {subscriptions.data.map((manga) => (
          <MangaRow
            key={manga.id}
            manga={manga}
            onClick={() => navigate({ name: "manga", mangaId: String(manga.id) })}
            action={
              <Button
                variant="ghost"
                size="icon"
                aria-label={t("subscriptions.unsubscribe")}
                disabled={unsubscribe.isPending && unsubscribe.variables === manga.id}
                onClick={() => unsubscribe.mutate(manga.id)}
              >
                <BookmarkX className="size-4" />
              </Button>
            }
          />
        ))}
      </div>
    )
  }

  return (
    <section className="space-y-3">
      <h1 className="text-lg font-semibold">{t("subscriptions.title")}</h1>
      {body}
    </section>
  )
}
