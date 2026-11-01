import { useEffect, useState } from "react"

/** Значение, обновляющееся с задержкой delayMs после последнего изменения
 * value — для debounce поискового ввода. */
export function useDebouncedValue<T>(value: T, delayMs: number): T {
  const [debounced, setDebounced] = useState(value)

  useEffect(() => {
    const timer = window.setTimeout(() => setDebounced(value), delayMs)
    return () => window.clearTimeout(timer)
  }, [value, delayMs])

  return debounced
}
