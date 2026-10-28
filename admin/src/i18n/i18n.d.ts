import type { defaultLocale } from "./index"
import type ru from "./ru.json"

// Ключи словаря выводятся из базового ru.json: t("nav.manga") типизирован,
// опечатка ловится на компиляции.
type Dictionary = typeof ru
export type MessageKey = FlattenKeys<Dictionary>

type FlattenKeys<T> = T extends string
  ? never
  : { [K in keyof T & string]: T[K] extends string ? K : `${K}.${FlattenKeys<T[K]>}` }[keyof T &
      string]

declare module "i18next" {
  interface CustomTypeOptions {
    defaultNS: "translation"
    resources: {
      translation: Dictionary
    }
    // базовый словарь — русский
    defaultLocale: typeof defaultLocale
  }
}
