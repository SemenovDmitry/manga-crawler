// Zod-схема поисковых параметров каталога манги: фильтры живут в URL.
import { z } from "zod"

import { PUBLICATION_STATUSES } from "@/types/models"

export const mangaSearchSchema = z.object({
  query: z.string().optional(),
  sourceId: z.coerce.number().int().positive().optional(),
  status: z.enum(PUBLICATION_STATUSES).optional(),
  limit: z.coerce.number().int().min(1).max(200).default(20),
  page: z.coerce.number().int().min(1).default(1),
})

export type MangaSearch = z.infer<typeof mangaSearchSchema>
