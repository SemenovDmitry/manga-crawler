// Форматирование дат для админки: локаль берём из i18n (ru/en).
import i18next from "i18next"

export function formatDateTime(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) {
    return iso
  }
  return new Intl.DateTimeFormat(i18next.language, {
    dateStyle: "short",
    timeStyle: "short",
  }).format(d)
}
