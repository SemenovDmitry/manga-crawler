import { useNavigate } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

import { PageSpinner } from "@/components/Spinner"
import { Button } from "@/components/ui/Button"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/Table"
import { StatusBadge } from "@/features/Manga"
import { formatDateTime } from "@/lib/format"
import type { IManga } from "@/types/models"
import { useChatUserSubscription } from "./queries"

// UserSubscription — подписки одного чата: заголовок с именем пользователя,
// таблица манг без пагинации (объём ограничен подписками чата).
export function UserSubscription({ chatId }: { chatId: number }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { data, isPending, isError } = useChatUserSubscription(chatId)

  if (isPending) {
    return <PageSpinner />
  }

  if (isError || !data) {
    return <p className="py-12 text-center text-destructive">{t("errors.notFound")}</p>
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Button variant="link" size="sm" onClick={() => navigate({ to: "/chat-users" })}>
          {t("userSubscription.back")}
        </Button>
      </div>
      <h1 className="text-xl font-semibold">{t("userSubscription.title", { chatId })}</h1>
      {data.manga.length === 0 ? (
        <p className="text-muted-foreground">{t("userSubscription.empty")}</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("manga.form.title")}</TableHead>
              <TableHead>{t("manga.source")}</TableHead>
              <TableHead>{t("manga.chaptersCount")}</TableHead>
              <TableHead>{t("manga.statusLabel")}</TableHead>
              <TableHead>{t("manga.lastCheck")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.manga.map((m) => (
              <SubscriptionRow key={m.id} manga={m} />
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  )
}

function SubscriptionRow({ manga }: { manga: IManga }) {
  const navigate = useNavigate()

  return (
    <TableRow
      className="cursor-pointer"
      onClick={() => void navigate({ to: "/manga/$id", params: { id: String(manga.id) } })}
    >
      <TableCell className="font-medium">{manga.title}</TableCell>
      <TableCell className="text-muted-foreground">{manga.source_name}</TableCell>
      <TableCell className="text-muted-foreground">{manga.chapters_count}</TableCell>
      <TableCell>
        <StatusBadge status={manga.publication_status} />
      </TableCell>
      <TableCell className="text-muted-foreground">
        {manga.last_check_at ? formatDateTime(manga.last_check_at) : "—"}
      </TableCell>
    </TableRow>
  )
}
