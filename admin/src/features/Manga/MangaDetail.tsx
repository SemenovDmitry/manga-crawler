import { useNavigate } from "@tanstack/react-router"
import { type ReactNode, useState } from "react"
import { useTranslation } from "react-i18next"

import { PageSpinner } from "@/components/Spinner"
import { Button } from "@/components/ui/Button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card"
import {
  Dialog,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/Dialog"
import { useSources } from "@/features/Sources"
import { formatDateTime } from "@/lib/format"
import { showToast } from "@/lib/toastStore"
import type { IManga, IMangaUpdate } from "@/types/models"
import { StatusBadge } from "./Manga"
import { MangaForm } from "./MangaForm"
import { useDeleteManga, useManga, useUpdateManga } from "./queries"
import { readerUrl } from "./readerUrl"

export function MangaDetail({ id }: { id: number }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { data: manga, isPending, isError } = useManga(id)
  const { data: sourcesData } = useSources()
  const update = useUpdateManga()
  const remove = useDeleteManga()

  const [editing, setEditing] = useState(false)
  const [confirmOpen, setConfirmOpen] = useState(false)

  if (isPending) {
    return <PageSpinner />
  }

  if (isError || !manga) {
    return <p className="py-12 text-center text-destructive">{t("errors.notFound")}</p>
  }

  if (editing) {
    return (
      <MangaForm
        mode="edit"
        sources={sourcesData?.sources ?? []}
        initial={manga}
        pending={update.isPending}
        onSubmit={(patch: IMangaUpdate) =>
          update.mutateAsync({ id, patch }).then(() => {
            showToast("success", t("manga.saved"))
            setEditing(false)
          })
        }
        onCancel={() => setEditing(false)}
      />
    )
  }

  return (
    <MangaCardView
      manga={manga}
      readerLink={readerUrl(manga, sourcesData?.sources ?? [])}
      deletePending={remove.isPending}
      confirmOpen={confirmOpen}
      onConfirmOpenChange={setConfirmOpen}
      onBack={() => navigate({ to: "/manga" })}
      onEdit={() => setEditing(true)}
      onDelete={() =>
        remove.mutate(id, {
          onSuccess: () => {
            setConfirmOpen(false)
            showToast("success", t("manga.deleted"))
            void navigate({ to: "/manga" })
          },
        })
      }
    />
  )
}

function MangaCardView({
  manga,
  readerLink,
  deletePending,
  confirmOpen,
  onConfirmOpenChange,
  onBack,
  onEdit,
  onDelete,
}: {
  manga: IManga
  readerLink: string | null
  deletePending: boolean
  confirmOpen: boolean
  onConfirmOpenChange: (open: boolean) => void
  onBack: () => void
  onEdit: () => void
  onDelete: () => void
}) {
  const { t } = useTranslation()

  return (
    <div className="space-y-4">
      <div>
        <Button variant="ghost" size="sm" onClick={onBack}>
          {t("manga.back")}
        </Button>
      </div>
      <Card className="max-w-2xl">
        <CardHeader>
          <CardTitle className="text-xl">{manga.title}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <DetailRow label={t("manga.source")}>{manga.source_name}</DetailRow>
          <DetailRow label={t("manga.form.url")}>
            {readerLink ? (
              // Ссылка на карточку у источника: base_url + url, внешняя
              // страница, поэтому target="_blank" оправдан.
              <a
                href={readerLink}
                target="_blank"
                rel="noreferrer"
                className="break-all text-primary underline-offset-4 hover:underline"
              >
                {readerLink}
              </a>
            ) : (
              <span className="break-all text-muted-foreground">—</span>
            )}
          </DetailRow>
          {manga.alt_titles && (
            <DetailRow label={t("manga.altTitles")}>{manga.alt_titles}</DetailRow>
          )}
          <DetailRow label={t("manga.latestChapter")}>{manga.latest_chapter || "—"}</DetailRow>
          <DetailRow label={t("manga.chaptersCount")}>{manga.chapters_count}</DetailRow>
          <DetailRow label={t("manga.statusLabel")}>
            <StatusBadge status={manga.publication_status} />
          </DetailRow>
          <DetailRow label={t("manga.lastCheck")}>
            {manga.last_check_at ? formatDateTime(manga.last_check_at) : "—"}
          </DetailRow>
          {manga.created_at && (
            <DetailRow label={t("chatUsers.createdAt")}>
              {formatDateTime(manga.created_at)}
            </DetailRow>
          )}
          {manga.updated_at && (
            <DetailRow label={t("manga.updatedAt")}>{formatDateTime(manga.updated_at)}</DetailRow>
          )}
          <div className="flex gap-2 pt-2">
            <Button onClick={onEdit}>{t("common.edit")}</Button>
            <Button variant="destructive" onClick={() => onConfirmOpenChange(true)}>
              {t("common.delete")}
            </Button>
          </div>
        </CardContent>
      </Card>

      <Dialog open={confirmOpen} onOpenChange={onConfirmOpenChange}>
        <DialogHeader>
          <DialogTitle>{t("manga.deleteTitle", { title: manga.title })}</DialogTitle>
          <DialogDescription>{t("manga.deleteDescription")}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button
            variant="ghost"
            disabled={deletePending}
            onClick={() => onConfirmOpenChange(false)}
          >
            {t("common.cancel")}
          </Button>
          <Button variant="destructive" disabled={deletePending} onClick={onDelete}>
            {t("common.delete")}
          </Button>
        </DialogFooter>
      </Dialog>
    </div>
  )
}

function DetailRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="grid grid-cols-[minmax(9rem,auto)_1fr] gap-3 text-sm">
      <span className="text-muted-foreground">{label}</span>
      <span>{children}</span>
    </div>
  )
}
