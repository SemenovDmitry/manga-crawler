import type { components, operations } from "@/api/schema.gen"

export type ISource =
  operations["listSources"]["responses"][200]["content"]["application/json"]["sources"][number]

export type IChatUser =
  operations["listChatUsers"]["responses"][200]["content"]["application/json"]["users"][number]

export type IManga = components["schemas"]["Manga"]
export type IMangaCreate = components["schemas"]["MangaCreate"]
export type IMangaUpdate = components["schemas"]["MangaUpdate"]
export type IPublicationStatus = components["schemas"]["PublicationStatus"]

// Все допустимые статусы публикации (для селектов и фильтров).
export const PUBLICATION_STATUSES: readonly IPublicationStatus[] = [
  "ongoing",
  "completed",
  "hiatus",
  "cancelled",
  "discontinued",
]
