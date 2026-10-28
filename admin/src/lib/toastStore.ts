// Минималистичный тост-стор без внешних зависимостей (sonner сознательно
// не тянули ради трёх сообщений). Подписка через useSyncExternalStore.

export type ToastKind = "error" | "success" | "info"

export interface Toast {
  id: number
  kind: ToastKind
  message: string
}

let nextId = 1
let toasts: Toast[] = []
const subscribers = new Set<() => void>()

function notify() {
  for (const cb of subscribers) {
    cb()
  }
}

export function subscribeToasts(cb: () => void): () => void {
  subscribers.add(cb)
  return () => {
    subscribers.delete(cb)
  }
}

export function getToasts(): Toast[] {
  return toasts
}

export function showToast(kind: ToastKind, message: string, ttlMs = 5000) {
  const toast: Toast = { id: nextId++, kind, message }
  toasts = [...toasts, toast]
  notify()
  setTimeout(() => dismissToast(toast.id), ttlMs)
}

export function dismissToast(id: number) {
  const before = toasts.length
  toasts = toasts.filter((t) => t.id !== id)
  if (toasts.length !== before) {
    notify()
  }
}
