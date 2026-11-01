import { useNavigate } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"
import { PageSpinner } from "@/components/Spinner"
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
import type { IChatUser } from "@/types/models"
import { useChatUsers, useSetChatUserEnabled } from "./queries"

export function ChatUsers() {
  const { t } = useTranslation()
  const { data, isPending } = useChatUsers()

  if (isPending) {
    return <PageSpinner />
  }

  return (
    <div className="space-y-4">
      <h1 className="text-xl font-semibold">{t("nav.chatUsers")}</h1>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t("chatUsers.username")}</TableHead>
            <TableHead>{t("chatUsers.chatId")}</TableHead>
            <TableHead>{t("chatUsers.createdAt")}</TableHead>
            <TableHead>{t("chatUsers.active")}</TableHead>
            <TableHead>{t("chatUsers.blocked")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {(data?.users ?? []).map((u) => (
            <ChatUserRow key={u.chat_id} user={u} />
          ))}
        </TableBody>
      </Table>
    </div>
  )
}

function ChatUserRow({ user }: { user: IChatUser }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const setEnabled = useSetChatUserEnabled()

  return (
    <TableRow
      className="cursor-pointer"
      onClick={() =>
        void navigate({ to: "/chat-users/$chatId", params: { chatId: String(user.chat_id) } })
      }
    >
      <TableCell className="font-medium">{user.username || "—"}</TableCell>
      <TableCell className="text-muted-foreground">{user.chat_id}</TableCell>
      <TableCell className="text-muted-foreground">
        {user.created_at ? formatDateTime(user.created_at) : "—"}
      </TableCell>
      <TableCell>{user.is_active ? t("chatUsers.yes") : t("chatUsers.no")}</TableCell>
      <TableCell>
        <div onClick={(e) => e.stopPropagation()}>
          <Switch
            checked={user.blocked}
            disabled={setEnabled.isPending}
            onChange={(e) => setEnabled.mutate({ chatId: user.chat_id, enabled: !e.target.checked })}
          />
        </div>
      </TableCell>
    </TableRow>
  )
}
