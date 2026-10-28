import type { ISource } from "@/types/models"

// Адрес карточки манги на источнике: base_url источника + url записи —
// так же адрес собирает бэкенд (backend/internal/telegram/mangaView.go).
// source_name — внутренний идентификатор, а не хост, поэтому для ссылки
// нужен base_url из списка источников.
export function readerUrl(
  manga: { source_id: number; url: string },
  sources: ISource[],
): string | null {
  const source = sources.find((s) => s.id === manga.source_id)
  if (!source?.base_url) {
    return null
  }
  return `${source.base_url.replace(/\/+$/, "")}/${manga.url.replace(/^\/+/, "")}`
}
