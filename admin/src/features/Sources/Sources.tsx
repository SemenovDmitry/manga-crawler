import { useState } from "react"
import { useTranslation } from "react-i18next"
import { PageSpinner, Spinner } from "@/components/Spinner"
import { Badge } from "@/components/ui/Badge"
import { Button } from "@/components/ui/Button"
import {
  Dialog,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/Dialog"
import { Switch } from "@/components/ui/Switch"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/Table"
import { formatDateTime } from "@/lib/format"
import { showToast } from "@/lib/toastStore"
import type { ISource } from "@/types/models"
import { useResetSourceFullCrawl, useSetSourceEnabled, useSources } from "./queries"

export function Sources() {
  const { t } = useTranslation()
  const { data, isPending } = useSources()

  if (isPending) {
    return <PageSpinner />
  }

  return (
    <div className="space-y-4">
      <h1 className="text-xl font-semibold">{t("nav.sources")}</h1>
      <SourcesTable sources={data?.sources ?? []} />
    </div>
  )
}

function SourcesTable({ sources }: { sources: ISource[] }) {
  const { t } = useTranslation()
  const setEnabled = useSetSourceEnabled()

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t("sources.name")}</TableHead>
          <TableHead>{t("sources.engine")}</TableHead>
          <TableHead>{t("sources.baseUrl")}</TableHead>
          <TableHead>{t("sources.lastFullCrawl")}</TableHead>
          <TableHead>{t("sources.enabled")}</TableHead>
          <TableHead>{t("sources.actions")}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {sources.map((s) => (
          <SourceRow
            key={s.id}
            source={s}
            onSetEnabled={setEnabled.mutate}
            pending={setEnabled.isPending}
          />
        ))}
      </TableBody>
    </Table>
  )
}

function SourceRow({
  source,
  onSetEnabled,
  pending,
}: {
  source: ISource
  onSetEnabled: (v: { id: number; enabled: boolean }) => void
  pending: boolean
}) {
  const { t } = useTranslation()

  return (
    <TableRow>
      <TableCell className="font-medium">{source.name}</TableCell>
      <TableCell>
        <Badge variant="secondary">{source.engine}</Badge>
      </TableCell>
      <TableCell className="text-muted-foreground">{source.base_url}</TableCell>
      <TableCell className="text-muted-foreground">
        {source.full_crawled_at ? formatDateTime(source.full_crawled_at) : t("sources.never")}
      </TableCell>
      <TableCell>
        <Switch
          checked={source.enabled}
          disabled={pending}
          onChange={(e) => onSetEnabled({ id: source.id, enabled: e.target.checked })}
        />
      </TableCell>
      <TableCell>
        <RecrawlButton source={source} />
      </TableCell>
    </TableRow>
  )
}

function RecrawlButton({ source }: { source: ISource }) {
  const { t } = useTranslation()
  const [confirmOpen, setConfirmOpen] = useState(false)
  const reset = useResetSourceFullCrawl()

  return (
    <>
      <Button
        variant="outline"
        size="sm"
        disabled={reset.isPending}
        onClick={() => setConfirmOpen(true)}
      >
        {reset.isPending ? <Spinner /> : null}
        {t("sources.recrawl")}
      </Button>
      <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <DialogHeader>
          <DialogTitle>{t("sources.recrawlTitle", { name: source.name })}</DialogTitle>
          <DialogDescription>{t("sources.recrawlDescription")}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="ghost" onClick={() => setConfirmOpen(false)}>
            {t("common.cancel")}
          </Button>
          <Button
            onClick={() => {
              reset.mutate(source.id, {
                onSuccess: () => {
                  setConfirmOpen(false)
                  showToast("success", t("sources.recrawlDone"))
                },
              })
            }}
          >
            {t("common.confirm")}
          </Button>
        </DialogFooter>
      </Dialog>
    </>
  )
}
