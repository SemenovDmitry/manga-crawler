import { useNavigate } from "@tanstack/react-router"
import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"

import { PageSpinner } from "@/components/Spinner"
import { Badge } from "@/components/ui/Badge"
import { Button } from "@/components/ui/Button"
import { Input } from "@/components/ui/Input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/Select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/Table"
import { useSources } from "@/features/Sources"
import { formatDateTime } from "@/lib/format"
import type { IManga, IPublicationStatus } from "@/types/models"
import { PUBLICATION_STATUSES } from "@/types/models"
import { useMangaList } from "./queries"
import type { MangaSearch } from "./search"

export interface MangaProps {
  search: MangaSearch
  onSearchChange: (patch: Partial<MangaSearch>) => void
}

// Бейдж статуса публикации: семантические варианты темы.
const STATUS_VARIANTS: Record<
  IPublicationStatus,
  "default" | "secondary" | "outline" | "destructive"
> = {
  ongoing: "default",
  completed: "secondary",
  hiatus: "outline",
  cancelled: "destructive",
  discontinued: "destructive",
}

export function StatusBadge({ status }: { status: IPublicationStatus }) {
  const { t } = useTranslation()
  return <Badge variant={STATUS_VARIANTS[status]}>{t(`manga.status.${status}`)}</Badge>
}

export function Manga({ search, onSearchChange }: MangaProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { data, isPending } = useMangaList(search)
  const { data: sourcesData } = useSources()

  // Поиск по названию с дебаунсом: печатаем локально, в URL уезжает через 300 мс.
  const [queryInput, setQueryInput] = useState(search.query ?? "")
  useEffect(() => {
    if (queryInput === (search.query ?? "")) {
      return
    }
    const timer = setTimeout(() => {
      onSearchChange({ query: queryInput.trim() || undefined, page: 1 })
    }, 300)
    return () => clearTimeout(timer)
  }, [queryInput, search.query, onSearchChange])

  const manga = data?.manga ?? []
  const total = data?.total ?? 0
  const pages = Math.max(1, Math.ceil(total / search.limit))

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-4">
        <h1 className="text-xl font-semibold">{t("manga.title")}</h1>
        <Button onClick={() => navigate({ to: "/manga/new" })}>{t("manga.create")}</Button>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Input
          className="w-full sm:max-w-xs"
          placeholder={t("manga.searchPlaceholder")}
          value={queryInput}
          onChange={(e) => setQueryInput(e.target.value)}
        />
        {/* value=null — вариант «все»: пустой фильтр не уходит в запрос. */}
        <Select
          items={[
            { value: null, label: t("manga.allSources") },
            ...(sourcesData?.sources ?? []).map((s) => ({ value: String(s.id), label: s.name })),
          ]}
          value={search.sourceId ? String(search.sourceId) : null}
          onValueChange={(value) =>
            onSearchChange({
              sourceId: value ? Number(value) : undefined,
              page: 1,
            })
          }
        >
          <SelectTrigger className="w-full sm:w-52">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={null}>{t("manga.allSources")}</SelectItem>
            {(sourcesData?.sources ?? []).map((s) => (
              <SelectItem key={s.id} value={String(s.id)}>
                {s.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select
          items={[
            { value: null, label: t("manga.allStatuses") },
            ...PUBLICATION_STATUSES.map((s) => ({
              value: s,
              label: t(`manga.status.${s}`),
            })),
          ]}
          value={search.status ?? null}
          onValueChange={(value) =>
            onSearchChange({
              // Вариант выбирается только из этого же списка статусов.
              status: value ? (value as IPublicationStatus) : undefined,
              page: 1,
            })
          }
        >
          <SelectTrigger className="w-full sm:w-44">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={null}>{t("manga.allStatuses")}</SelectItem>
            {PUBLICATION_STATUSES.map((s) => (
              <SelectItem key={s} value={s}>
                {t(`manga.status.${s}`)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {isPending ? (
        <PageSpinner />
      ) : manga.length === 0 ? (
        <p className="py-12 text-center text-muted-foreground">{t("manga.empty")}</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-16">ID</TableHead>
              <TableHead>{t("manga.form.title")}</TableHead>
              <TableHead>{t("manga.source")}</TableHead>
              <TableHead>{t("manga.latestChapter")}</TableHead>
              <TableHead>{t("manga.chaptersCount")}</TableHead>
              <TableHead>{t("manga.statusLabel")}</TableHead>
              <TableHead>{t("manga.lastCheck")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {manga.map((m) => (
              <MangaRow
                key={m.id}
                manga={m}
                onOpen={() => navigate({ to: "/manga/$id", params: { id: String(m.id) } })}
              />
            ))}
          </TableBody>
        </Table>
      )}

      <div className="flex items-center justify-end gap-2">
        <span className="text-sm text-muted-foreground">
          {t("manga.pageOf", { page: search.page, pages })}
        </span>
        <Button
          variant="outline"
          size="sm"
          disabled={search.page <= 1}
          onClick={() => onSearchChange({ page: search.page - 1 })}
        >
          {t("manga.prev")}
        </Button>
        <Button
          variant="outline"
          size="sm"
          disabled={search.page >= pages}
          onClick={() => onSearchChange({ page: search.page + 1 })}
        >
          {t("manga.next")}
        </Button>
      </div>
    </div>
  )
}

function MangaRow({ manga, onOpen }: { manga: IManga; onOpen: () => void }) {
  return (
    // Клик по строке — переход к карточке; контент внутри не интерактивный,
    // поэтому достаточно cursor-pointer.
    <TableRow className="cursor-pointer" onClick={onOpen}>
      <TableCell className="text-muted-foreground">{manga.id}</TableCell>
      <TableCell>
        <div className="font-medium">{manga.title}</div>
        {manga.alt_titles && (
          <div className="text-xs text-muted-foreground">{manga.alt_titles}</div>
        )}
      </TableCell>
      <TableCell className="text-muted-foreground">{manga.source_name}</TableCell>
      <TableCell>{manga.latest_chapter || "—"}</TableCell>
      <TableCell>{manga.chapters_count}</TableCell>
      <TableCell>
        <StatusBadge status={manga.publication_status} />
      </TableCell>
      <TableCell className="text-muted-foreground">
        {manga.last_check_at ? formatDateTime(manga.last_check_at) : "—"}
      </TableCell>
    </TableRow>
  )
}
