import { useTranslation } from "react-i18next"

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/Select"
import { changeLocale, type Locale, locales } from "@/i18n"

export function LocaleSwitch() {
  const { t, i18n } = useTranslation()
  const items: Array<{ value: Locale; label: string }> = locales.map((l) => ({
    value: l,
    label: l.toUpperCase(),
  }))

  return (
    <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
      <span>{t("common.language")}</span>
      <Select
        items={items}
        value={i18n.language}
        onValueChange={(value) => changeLocale(value as Locale)}
      >
        <SelectTrigger size="sm" className="w-[4.5rem]" aria-label={t("common.language")}>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {items.map((item) => (
            <SelectItem key={item.value} value={item.value}>
              {item.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </span>
  )
}
