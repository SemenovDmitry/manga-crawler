import { Bookmark, CircleQuestionMark, type LucideIcon, Search } from "lucide-react"
import { useTranslation } from "react-i18next"

import type { MessageKey } from "@/i18n"
import { navigate } from "@/lib/router"
import { cn } from "@/lib/utils"

export type TabName = "search" | "subscriptions" | "help"

const TABS: Array<{ name: TabName; labelKey: MessageKey; Icon: LucideIcon }> = [
  { name: "search", labelKey: "nav.search", Icon: Search },
  { name: "subscriptions", labelKey: "nav.subscriptions", Icon: Bookmark },
  { name: "help", labelKey: "nav.help", Icon: CircleQuestionMark },
]

export function BottomNav({ active }: { active: TabName }) {
  const { t } = useTranslation()

  return (
    <nav className="sticky bottom-0 z-10 flex border-t bg-background/95 backdrop-blur">
      {TABS.map(({ name, labelKey, Icon }) => (
        <button
          key={name}
          type="button"
          onClick={() => navigate({ name })}
          aria-current={active === name ? "page" : undefined}
          className={cn(
            "flex flex-1 cursor-pointer flex-col items-center gap-1 py-2 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground",
            active === name && "text-primary",
          )}
        >
          <Icon className="size-5" />
          <span>{t(labelKey)}</span>
        </button>
      ))}
    </nav>
  )
}
