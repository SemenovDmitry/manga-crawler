import type { components, operations } from "@/api/schema.gen"

export type Source =
  operations["listSources"]["responses"][200]["content"]["application/json"]["sources"][number]

export type ChatUser =
  operations["listChatUsers"]["responses"][200]["content"]["application/json"]["users"][number]

export type Manga = components["schemas"]["Manga"]
export type MangaCreate = components["schemas"]["MangaCreate"]
export type MangaUpdate = components["schemas"]["MangaUpdate"]
export type PublicationStatus = components["schemas"]["PublicationStatus"]

// Все допустимые статусы публикации (для селектов и фильтров).
export const PUBLICATION_STATUSES: readonly PublicationStatus[] = [
  "ongoing",
  "completed",
  "hiatus",
  "cancelled",
  "discontinued",
]
