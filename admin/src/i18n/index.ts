import i18n from "i18next"
import { initReactI18next } from "react-i18next"

import en from "./en.json"
import ru from "./ru.json"

// RU — базовый язык; EN — второй. Типизация ключей выводится из ru.json,
// опечатка в ключе ловится на компиляции (см. i18n.d.ts).
export const defaultLocale = "ru"
export const locales = ["ru", "en"] as const
export type Locale = (typeof locales)[number]

const stored = localStorage.getItem("admin-locale")
const initialLocale: Locale = locales.includes(stored as Locale)
  ? (stored as Locale)
  : defaultLocale

void i18n.use(initReactI18next).init({
  resources: {
    ru: { translation: ru },
    en: { translation: en },
  },
  lng: initialLocale,
  fallbackLng: defaultLocale,
  interpolation: {
    escapeValue: false, // React сам экранирует
  },
})

export function changeLocale(locale: Locale) {
  localStorage.setItem("admin-locale", locale)
  void i18n.changeLanguage(locale)
}
