import { useTranslation } from "react-i18next"

import { errorToMessage } from "@/api/errorMessages"

// Показать тост по ошибке API: известные статусы — локализованный текст,
// неизвестные — сообщение бэкенда как есть.
export function useApiToast() {
  const { t } = useTranslation()

  return (err: unknown) => {
    // Ленивый импорт исключил бы круговую зависимость; тосты — UI, их вызов
    // из этого хука допустим.
    import("@/lib/toastStore").then(({ showToast }) => {
      const { key, text } = errorToMessage(err)
      if (key) {
        showToast("error", t(key as never))
      } else if (text) {
        showToast("error", text)
      }
    })
  }
}
