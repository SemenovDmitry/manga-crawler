import { useTranslation } from "react-i18next"

import { ApiError } from "@/api/client"
import type { MangaDetail as MangaDetailData } from "@/api/types"
import { ErrorState } from "@/components/ErrorState"
import { StatusBadge } from "@/components/StatusBadge"
import { Button } from "@/components/ui/Button"
import { Card, CardContent } from "@/components/ui/Card"
import { Skeleton } from "@/components/ui/Skeleton"
import { useManga, useSubscribe, useUnsubscribe } from "./queries"

function DetailSkeleton() {
  return (
    <div className="space-y-4" aria-hidden>
      <Skeleton className="h-6 w-3/4" />
      <div className="flex gap-2">
        <Skeleton className="h-5 w-24" />
        <Skeleton className="h-5 w-28" />
      </div>
      <Skeleton className="h-4 w-1/2" />
      <Skeleton className="h-9 w-full" />
    </div>
  )
}

function DetailBody({ data, mangaId }: { data: MangaDetailData; mangaId: string }) {
  const { t } = useTranslation()
  const subscribe = useSubscribe(mangaId)
  const unsubscribe = useUnsubscribe(mangaId)
  const togglePending = subscribe.isPending || unsubscribe.isPending

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-semibold">{data.manga.title}</h1>
      {data.manga.alt_titles !== "" && (
        <p className="text-sm text-muted-foreground">{data.manga.alt_titles}</p>
      )}
      <div className="flex flex-wrap items-center gap-2 text-sm">
        <StatusBadge status={data.manga.publication_status} />
        <span className="text-muted-foreground">
          {t("manga.chapters", { count: data.manga.chapters_count })}
        </span>
        {data.manga.latest_chapter !== "" && (
          <span className="text-muted-foreground">
            · {t("manga.latestChapter", { chapter: data.manga.latest_chapter })}
          </span>
        )}
      </div>
      <p className="text-sm">
        <span className="text-muted-foreground">{t("manga.source")}: </span>
        <a
          href={data.manga.source_url}
          target="_blank"
          rel="noreferrer"
          className="text-link underline-offset-2 hover:underline"
        >
          {data.manga.source_name}
        </a>
      </p>
      <div className="space-y-2">
        <Button
          className="w-full"
          variant={data.subscribed ? "destructive" : "default"}
          disabled={togglePending}
          onClick={() => (data.subscribed ? unsubscribe.mutate() : subscribe.mutate())}
        >
          {data.subscribed ? t("manga.unsubscribe") : t("manga.subscribe")}
        </Button>
        {data.subscribed && (
          <p className="text-center text-xs text-muted-foreground">{t("manga.subscribed")}</p>
        )}
      </div>
    </div>
  )
}

/** Карточка манги: данные источника и кнопка подписки/отписки. */
export function MangaDetail({ mangaId }: { mangaId: string }) {
  const { t } = useTranslation()
  const detail = useManga(mangaId)

  if (detail.isPending) {
    return <DetailSkeleton />
  }
  if (detail.isError) {
    // 404 — контекстный текст «манга не найдена», прочие ошибки — общий блок.
    if (detail.error instanceof ApiError && detail.error.status === 404) {
      return (
        <div role="alert" className="py-8 text-center text-sm text-muted-foreground">
          {t("manga.notFound")}
        </div>
      )
    }
    return <ErrorState error={detail.error} onRetry={() => void detail.refetch()} />
  }

  return (
    <section>
      <Card>
        <CardContent className="pt-6">
          <DetailBody data={detail.data} mangaId={mangaId} />
        </CardContent>
      </Card>
    </section>
  )
}
