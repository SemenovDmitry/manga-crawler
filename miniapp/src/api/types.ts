import type { components } from "./schema.gen"

// Алиасы сгенерированных схем API. Ручные типы, дублирующие спеку
// backend/docs/openapi-miniapp.yaml, запрещены — перегенерация: npm run gen:api.
export type User = components["schemas"]["User"]
export type Manga = components["schemas"]["Manga"]
export type MangaList = components["schemas"]["MangaList"]
export type MangaDetail = components["schemas"]["MangaDetail"]
export type SubscriptionState = components["schemas"]["SubscriptionState"]
export type PublicationStatus = components["schemas"]["PublicationStatus"]
