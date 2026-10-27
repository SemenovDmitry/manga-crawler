import type { operations } from "@/api/schema.gen"

export type ISource =
  operations["listSources"]["responses"][200]["content"]["application/json"]["sources"][number]

export type IChatUser =
  operations["listChatUsers"]["responses"][200]["content"]["application/json"]["users"][number]
