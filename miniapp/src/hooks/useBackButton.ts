import { backButton } from "@telegram-apps/sdk-react"
import { useEffect } from "react"

/**
 * Показывает системную кнопку «Назад» Telegram, когда активна вложенная
 * страница (карточка манги), и вешает на неё обратный переход.
 * На платформах без BackButton — тихо ничего не делает.
 */
export function useBackButton(active: boolean, onBack: () => void) {
  useEffect(() => {
    if (!active) {
      return
    }
    let offClick: (() => void) | undefined
    try {
      backButton.show()
      const remove = backButton.onClick(() => onBack())
      offClick = () => remove()
    } catch {
      // BackButton недоступен (платформа/клиент без поддержки).
      return
    }
    return () => {
      offClick?.()
      try {
        backButton.hide()
      } catch {
        // уже скрыт/недоступен
      }
    }
  }, [active, onBack])
}
